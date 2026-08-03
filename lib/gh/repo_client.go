// Package gh wraps the GitHub REST and GraphQL APIs for syncing issues, PRs, and project items.
package gh

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/go-github/v89/github"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/katbyte/ghp-sync/lib/clog"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

func (t Token) NewClient() (*github.Client, context.Context) {
	ctx := context.Background()

	// use retryablehttp to handle rate limiting
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 7
	retryClient.Logger = clog.Log

	// github is.. special using 403 instead of 429 for rate limiting so we need to handle that here :(
	retryClient.Backoff = func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
		if resp != nil && resp.StatusCode == http.StatusForbidden {
			// get x-rate-limit-reset header
			reset := resp.Header.Get("X-Ratelimit-Reset")
			if reset != "" {
				i, err := strconv.ParseInt(reset, 10, 64)
				if err == nil {
					utime := time.Unix(i, 0)
					wait := time.Until(utime) + time.Minute // add an extra min to be safe
					clog.Log.Errorf("ratelimited, parsed x-ratelimit-reset, waiting for %s", wait.String())
					return wait
				}
				clog.Log.Errorf("unable to parse x-ratelimit-reset header: %s", err)
			}
		}

		return retryablehttp.DefaultBackoff(min, max, attemptNum, resp)
	}
	retryClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if err != nil {
			return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
		}
		if resp != nil && resp.StatusCode == http.StatusForbidden {
			return true, nil
		}

		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}

	if t := t.Token; t != nil {
		src := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: *t},
		)
		retryClient.HTTPClient = oauth2.NewClient(ctx, src)
	}

	// Wrap via StandardClient so the retryable transport stays in the chain
	httpClient := retryClient.StandardClient()

	client, err := github.NewClient(github.WithHTTPClient(httpClient))
	if err != nil {
		// only possible with invalid options such as a nil http client
		clog.Log.Fatalf("failed to create github client: %s", err)
	}

	return client, ctx
}

// NewGraphQLClient returns a githubv4 client with rate limit aware retries.
// todo we may want to update the above retry logic to match this one
func (t Token) NewGraphQLClient() (*githubv4.Client, context.Context, error) {
	ctx := context.Background()
	if t.Token == nil {
		return nil, ctx, errors.New("no GitHub token provided")
	}

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 7
	retryClient.Logger = clog.Log
	// Optional: tighten/loosen as you like; Backoff can override this.
	retryClient.RetryWaitMin = 2 * time.Second
	retryClient.RetryWaitMax = 60 * time.Second

	// Backoff that respects GitHub headers on 403/429
	retryClient.Backoff = func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
		// Prefer Retry-After when present (seconds)
		if resp != nil {
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, err := strconv.Atoi(ra); err == nil {
					wait := time.Duration(secs) * time.Second
					clog.Log.Errorf("ratelimited (Retry-After), waiting for %s", wait)
					return wait
				}
			}
			// GitHub primary limit: X-RateLimit-Reset (unix seconds)
			if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
				if reset := resp.Header.Get("X-Ratelimit-Reset"); reset != "" {
					if i, err := strconv.ParseInt(reset, 10, 64); err == nil {
						utime := time.Unix(i, 0)
						wait := time.Until(utime) + time.Minute // pad a minute to be safe
						if wait > 0 {
							clog.Log.Errorf("ratelimited (Reset header), waiting for %s", wait)
							return wait
						}
					} else {
						clog.Log.Errorf("unable to parse X-RateLimit-Reset: %v", err)
					}
				}
			}
		}
		// Fallback to exponential backoff with jitter
		return retryablehttp.DefaultBackoff(min, max, attemptNum, resp)
	}

	// Retry policy: 5xx, 429, and GitHub’s 403 secondary/abuse limits
	retryClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		// Network errors: let default policy decide
		if err != nil {
			return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
		}
		if resp == nil {
			return false, nil
		}

		// Standard: retry 5xx and 429
		if resp.StatusCode == 0 || resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			return true, nil
		}

		// GitHub quirk: secondary/abuse rate limits return 403
		if resp.StatusCode == http.StatusForbidden {
			// If remaining is 0, or Retry-After present, or we just decide to be safe—retry.
			if resp.Header.Get("X-Ratelimit-Remaining") == "0" ||
				resp.Header.Get("Retry-After") != "" {
				return true, nil
			}
			// Many secondary limits don’t set remaining=0. Be permissive and retry 403s.
			return true, nil
		}

		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}

	// OAuth2 bearer on the underlying HTTP client
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: *t.Token})
	retryClient.HTTPClient = oauth2.NewClient(ctx, src)

	// Pass the standard *http.Client into githubv4
	httpClient := retryClient.StandardClient()
	return githubv4.NewClient(httpClient), ctx, nil
}
