package gh

import (
	"context"
	"fmt"
	"github.com/google/go-github/v45/github"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/katbyte/ghp-sync/lib/clog"
	"github.com/katbyte/ghp-sync/lib/pointer"
	"golang.org/x/oauth2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Token struct {
	Token *string
}

type Repo struct {
	Owner string
	Name  string
	Token
}

func NewRepo(repo, token string) (*Repo, error) {
	parts := strings.Split(repo, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format, expected owner/name got %q", repo)
	}

	return pointer.To(NewRepoOwnerName(parts[0], parts[1], token)), nil
}

func NewRepoOwnerName(owner, name, token string) Repo {
	r := Repo{
		Owner: owner,
		Name:  name,
		Token: Token{
			Token: nil,
		},
	}

	if token != "" {
		r.Token.Token = &token
	}

	return r
}

func (t Token) NewClient() (*github.Client, context.Context) {
	ctx := context.Background()

	// use retryablehttp to handle rate limiting
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 7
	retryClient.Logger = clog.Log

	// github is.. special using 403 instead of 429 for rate limiting so we need to handle that here :(
	retryClient.Backoff = func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
		if resp != nil && resp.StatusCode == 403 {
			// get x-rate-limit-reset header
			reset := resp.Header.Get("x-ratelimit-reset")
			if reset != "" {
				i, err := strconv.ParseInt(reset, 10, 64)
				if err == nil {
					utime := time.Unix(i, 0)
					wait := utime.Sub(time.Now()) + time.Minute // add an extra min to be safe
					clog.Log.Errorf("ratelimited, parsed x-ratelimit-reset, waiting for %s", wait.String())
					return wait
				}
				clog.Log.Errorf("unable to parse x-ratelimit-reset header: %s", err)
			}
		}

		return retryablehttp.DefaultBackoff(min, max, attemptNum, resp)
	}
	retryClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if resp.StatusCode == 403 {
			return true, nil
		}

		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}

	if t := t.Token; t != nil {
		t := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: *t},
		)
		retryClient.HTTPClient = oauth2.NewClient(ctx, t)
	}

	return github.NewClient(retryClient.StandardClient()), ctx
}

type PRApproval struct {
	Data struct {
		Repository struct {
			PullRequest struct {
				Title          string
				ReviewDecision string
			}
		}
	}
}

func (r Repo) PRReviewDecision(pr int) (*string, error) {
	q := `query=
        query($owner: String!, $repo: String!, $pr: Int!) {
            repository(name: $repo, owner: $owner) {
                pullRequest(number: $pr) {
                    title
                    reviewDecision
                    state
                    reviews(first: 100) {
                        nodes {
                            state
                            author {
                                login
                            }
                        }
                    }
                }
            }
        }
    `

	p := [][]string{
		{"-f", "owner=" + r.Owner},
		{"-f", "repo=" + r.Name},
		{"-F", "pr=" + strconv.Itoa(pr)},
	}

	var approved PRApproval
	if err := r.GraphQLQueryUnmarshal(q, p, &approved); err != nil {
		return nil, err
	}

	return &approved.Data.Repository.PullRequest.ReviewDecision, nil
}
