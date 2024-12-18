package j

import (
	"fmt"
	"sort"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

const IssuePageSize = 50

// async list all issues for a jql with a callback per api request
func (i Instance) ListAllIssues(jql string, fields, expand *[]string, cb func(*models.IssueSearchScheme, *models.ResponseScheme) error) error {
	client, ctx, err := i.NewClient()
	if err != nil {
		return fmt.Errorf("unable to get client for '%s': %w", i.URL, err)
	}

	startAt := 0
	for {
		result, resp, err := client.Issue.Search.Get(ctx, jql, *fields, *expand, startAt, IssuePageSize, "true")
		if err != nil {
			return err
		}

		if err = cb(result, resp); err != nil {
			return fmt.Errorf("callback failed for %s @ %s (StartAt %d): %w", i.URL, jql, startAt, err)
		}

		if startAt > result.Total {
			// This means we have fetched all issues
			break
		}

		// Update the startAt for the next iteration
		startAt += IssuePageSize
	}

	return nil
}

// mmake multiple calls to get all issues for jql
func (i Instance) GetAllIssues(jql string, fields, expand *[]string) (*[]models.IssueScheme, error) {
	var allIssues []models.IssueScheme

	err := i.ListAllIssues(jql, fields, expand, func(results *models.IssueSearchScheme, resp *models.ResponseScheme) error {
		for _, i := range results.Issues {
			allIssues = append(allIssues, *i)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get all issues for %s @ %s: %w", i.URL, jql, err)
	}

	sort.Slice(allIssues, func(i, j int) bool {
		return allIssues[i].Key < allIssues[j].Key
	})

	return &allIssues, nil
}
