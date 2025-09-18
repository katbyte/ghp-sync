package cli

import (
	"context"
	"fmt"
	"time"

	c "github.com/gookit/color"
	"github.com/katbyte/ghp-sync/lib/gh"
	"github.com/spf13/cobra"
)

func humanReset(reset int) string {
	when := time.Unix(int64(reset), 0).Local()
	d := time.Until(when)
	if d < 0 {
		return "reset"
	}
	return d.Round(time.Second).String()
}

func CmdRateLimit(_ *cobra.Command, _ []string) error {
	f := GetFlags()
	r, err := gh.GetRateLimit(context.Background(), f.Token)
	if err != nil {
		return fmt.Errorf("unable to get rate limits: %w", err)
	}

	c.Printf("GitHub rate limits (local now: <lightCyan>%s</>):\n", time.Now().Format(time.RFC3339))

	type named struct {
		name string
		rate gh.Rate
	}
	ordered := []named{
		{"core", r.Core},
		{"graphql", r.GraphQL},
		{"search", r.Search},
		{"code_scanning_upload", r.CodeScanning},
		{"actions_runner_registration", r.ActionsRunnerRegistration},
		{"source_import", r.SourceImport},
		{"integration_manifest", r.IntegrationManifest},
		{"scim", r.Scim},
	}

	// Known buckets
	for _, it := range ordered {
		if it.rate.Limit == 0 && it.rate.Remaining == 0 && it.rate.Used == 0 && it.rate.Reset == 0 {
			continue
		}
		c.Printf("  <lightBlue>%-28s</> limit=<yellow>%5d</>  remaining=<green>%5d</>  used=<magenta>%5d</>  resets in <cyan>%s</>\n",
			it.name, it.rate.Limit, it.rate.Remaining, it.rate.Used, humanReset(it.rate.Reset))
	}

	// Any extra buckets
	for name, r := range r.Other {
		c.Printf("  <lightBlue>%-28s</> limit=<yellow>%5d</>  remaining=<green>%5d</>  used=<magenta>%5d</>  resets in <cyan>%s</>\n",
			name, r.Limit, r.Remaining, r.Used, humanReset(r.Reset))
	}

	// Alias
	if r.Core.Limit != 0 {
		c.Printf("  <blue>rate (alias of core)</>:        limit=<yellow>%5d</>  remaining=<green>%5d</>  used=<magenta>%5d</>  resets in <cyan>%s</>\n",
			r.Rate.Limit, r.Rate.Remaining, r.Rate.Used, humanReset(r.Rate.Reset))
	}

	return nil
}
