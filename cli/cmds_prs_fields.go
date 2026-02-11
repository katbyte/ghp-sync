package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/katbyte/ghp-sync/lib/gh"
)

// PRFieldContext holds all the data needed to compute field values
type PRFieldContext struct {
	PR          *gh.PullRequest
	Project     gh.Project
	DaysOpen    int
	DaysWaiting int
	Status      string // The computed status text (e.g., "In Progress", "Approved")
}

// PRFieldDef defines a field that can be populated on a GitHub Project item
type PRFieldDef struct {
	Type      gh.ItemValueType // Field type for GraphQL mutation
	ComputeFn func(ctx PRFieldContext) interface{}
}

// PRFields is the registry of all available PR fields, keyed by field name (matches GitHub Project field name)
var PRFields = map[string]PRFieldDef{
	"PR#": {
		Type: gh.ItemValueTypeNumber,
		ComputeFn: func(ctx PRFieldContext) interface{} {
			return ctx.PR.Number
		},
	},
	"Status": {
		Type: gh.ItemValueTypeSingleSelect,
		ComputeFn: func(ctx PRFieldContext) interface{} {
			id, ok := ctx.Project.StatusIDs[ctx.Status]
			if !ok || id == "" {
				fmt.Printf("WARNING: status %q not found in project\n", ctx.Status)
				return nil
			}
			return id
		},
	},
	"User": {
		Type: gh.ItemValueTypeText,
		ComputeFn: func(ctx PRFieldContext) interface{} {
			return ctx.PR.Author
		},
	},
	"Open Days": {
		Type: gh.ItemValueTypeNumber,
		ComputeFn: func(ctx PRFieldContext) interface{} {
			return ctx.DaysOpen
		},
	},
	"Waiting Days": {
		Type: gh.ItemValueTypeNumber,
		ComputeFn: func(ctx PRFieldContext) interface{} {
			return ctx.DaysWaiting
		},
	},
	"Comment Count": {
		Type: gh.ItemValueTypeNumber,
		ComputeFn: func(ctx PRFieldContext) interface{} {
			return ctx.PR.TotalCommentCount
		},
	},
	"Review Count": {
		Type: gh.ItemValueTypeNumber,
		ComputeFn: func(ctx PRFieldContext) interface{} {
			return ctx.PR.TotalReviewCount
		},
	},
	"Review Comment Count": {
		Type: gh.ItemValueTypeNumber,
		ComputeFn: func(ctx PRFieldContext) interface{} {
			return ctx.PR.ReviewCommentCount
		},
	},
	"Created At": {
		Type: gh.ItemValueTypeDate,
		ComputeFn: func(ctx PRFieldContext) interface{} {
			return ctx.PR.CreatedAt.Format(time.RFC3339)
		},
	},
	"Closed At": {
		Type: gh.ItemValueTypeDate,
		ComputeFn: func(ctx PRFieldContext) interface{} {
			if strings.EqualFold(ctx.PR.State, "open") {
				return nil // Don't set for open PRs
			}
			return ctx.PR.ClosedAt.Format(time.RFC3339)
		},
	},
	"Filtered Review Count": {
		Type: gh.ItemValueTypeNumber,
		ComputeFn: func(ctx PRFieldContext) interface{} {
			if ctx.PR.FilteredReviewCount == 0 {
				return nil
			}
			return ctx.PR.FilteredReviewCount
		},
	},
	"Filtered Review Comment Count": {
		Type: gh.ItemValueTypeNumber,
		ComputeFn: func(ctx PRFieldContext) interface{} {
			if ctx.PR.FilteredReviewCount == 0 {
				return nil
			}
			return ctx.PR.FilteredReviewCommentCount
		},
	},
}
