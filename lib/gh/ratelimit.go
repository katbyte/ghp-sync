package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Rate struct {
	Limit     int `json:"limit"`
	Used      int `json:"used"`
	Remaining int `json:"remaining"`
	Reset     int `json:"reset"` // epoch seconds
}

// Flat, no "resources" nesting
type RateLimits struct {
	Core                      Rate
	GraphQL                   Rate
	Search                    Rate
	SourceImport              Rate
	IntegrationManifest       Rate
	CodeScanning              Rate // code_scanning_upload
	ActionsRunnerRegistration Rate
	Scim                      Rate

	// "rate" (alias for core)
	Rate Rate

	// Any new/unknown buckets GitHub adds in the future
	Other map[string]Rate
}

func GetRateLimit(ctx context.Context, token string) (*RateLimits, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/rate_limit", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("rate_limit request failed: %s\n%s", resp.Status, body)
	}

	// Raw decoding target
	var raw struct {
		Resources map[string]Rate `json:"resources"`
		Rate      Rate            `json:"rate"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	rl := &RateLimits{
		Other: map[string]Rate{},
		Rate:  raw.Rate,
	}

	// Assign known buckets
	if r, ok := raw.Resources["core"]; ok {
		rl.Core = r
	}
	if r, ok := raw.Resources["graphql"]; ok {
		rl.GraphQL = r
	}
	if r, ok := raw.Resources["search"]; ok {
		rl.Search = r
	}
	if r, ok := raw.Resources["source_import"]; ok {
		rl.SourceImport = r
	}
	if r, ok := raw.Resources["integration_manifest"]; ok {
		rl.IntegrationManifest = r
	}
	if r, ok := raw.Resources["code_scanning_upload"]; ok {
		rl.CodeScanning = r
	}
	if r, ok := raw.Resources["actions_runner_registration"]; ok {
		rl.ActionsRunnerRegistration = r
	}
	if r, ok := raw.Resources["scim"]; ok {
		rl.Scim = r
	}

	// Anything else goes to Other
	for k, v := range raw.Resources {
		switch k {
		case "core", "graphql", "search", "source_import", "integration_manifest",
			"code_scanning_upload", "actions_runner_registration", "scim":
			continue
		default:
			rl.Other[k] = v
		}
	}

	return rl, nil
}
