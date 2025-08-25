package gh

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ParseGitHubURL parses a GitHub PR/Issue URL and returns the owner, repo name, type, and number.
func ParseGitHubURL(gitHubURL string) (owner string, name string, typ string, number int, err error) {
	parsedURL, err := url.Parse(gitHubURL)
	if err != nil {
		return "", "", "", 0, fmt.Errorf("invalid URL: %w", err)
	}

	// Check if the URL is from GitHub
	if !strings.Contains(parsedURL.Host, "github.com") {
		return "", "", "", 0, errors.New("URL is not a GitHub URL")
	}

	// Split the path and validate it
	segments := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(segments) < 4 {
		return "", "", "", 0, errors.New("URL path is not in the expected format")
	}

	// Extract details
	owner = segments[0]
	name = segments[1]
	typ = segments[2]
	if typ != "pull" && typ != "issues" {
		return "", "", "", 0, errors.New("URL type is neither a pull request nor an issue")
	}

	// Parse the number
	var numberVal int
	if _, err = fmt.Sscanf(segments[3], "%d", &numberVal); err != nil {
		return "", "", "", 0, fmt.Errorf("failed to parse number: %w", err)
	}
	number = numberVal

	return owner, name, typ, number, nil
}
