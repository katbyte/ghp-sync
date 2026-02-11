package gh

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
	"time"
)

func (t Token) GraphQLQueryUnmarshal(query string, params [][]string, data interface{}) error {
	out, err := t.GraphQLQuery(query, params)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(*out), data)
}

func (t Token) GraphQLQuery(query string, params [][]string) (*string, error) {
	const (
		maxAttempts = 5
		baseDelay   = time.Minute
	)

	args := []string{"api", "graphql", "-f", query}

	for _, p := range params {
		args = append(args, p[0])
		args = append(args, p[1])
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ghc := exec.Command("gh", args...) //nolint:gosec // args are constructed internally

		// Preserve existing environment and add GITHUB_TOKEN if present
		env := os.Environ()
		if t.Token != nil {
			env = append(env, "GITHUB_TOKEN="+*t.Token)
		}
		ghc.Env = env

		out, err := ghc.CombinedOutput()
		outstr := string(out)

		// Success: return immediately
		if err == nil {
			return &outstr, nil
		}

		// If it doesn't look like a rate limit error, fail fast
		if !isRateLimitError(outstr) && !isRateLimitError(err.Error()) {
			return nil, fmt.Errorf("gh graphql failed: %w\noutput: %s", err, outstr)
		}

		// If we've hit rate limit and used all attempts, bail out
		if attempt == maxAttempts {
			return nil, fmt.Errorf("rate limited after %d attempts: %w\nlast output: %s", attempt, err, outstr)
		}

		// Exponential backoff (1s, 2s, 4s, 8s, ...)
		delay := baseDelay * time.Duration(math.Pow(2, float64(attempt-1)))
		time.Sleep(delay)
	}

	// Should be unreachable, but keeps compiler happy
	return nil, errors.New("gh graphql failed after retries, this should be unreachable")
}

func isRateLimitError(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "rate limit") ||
		strings.Contains(m, "api rate limit exceeded") ||
		strings.Contains(m, "secondary rate limit") ||
		strings.Contains(m, "abuse detection")
}
