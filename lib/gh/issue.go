package gh

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/google/go-github/v89/github"
	"github.com/katbyte/go-kt/clog"
)

func (r Repo) ListAllIssues(state string, cb func([]*github.Issue, *github.Response) error) error {
	client, ctx := r.NewClient()
	opts := &github.IssueListByRepoOptions{
		State: state,
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}

	for {
		clog.Log.Debugf("Listing all Issues for %s/%s (Page %d)...", r.Owner, r.Name, opts.ListOptions.Page)
		issues, resp, err := client.Issues.ListByRepo(ctx, r.Owner, r.Name, opts)
		if err != nil {
			return fmt.Errorf("unable to list Issues for %s/%s (Page %d): %w", r.Owner, r.Name, opts.ListOptions.Page, err)
		}

		if err = cb(issues, resp); err != nil {
			return fmt.Errorf("callback failed for %s/%s (Page %d): %w", r.Owner, r.Name, opts.ListOptions.Page, err)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = resp.NextPage
	}

	return nil
}

func (r Repo) GetIssue(number int) (*github.Issue, error) {
	client, ctx := r.NewClient()

	i, _, err := client.Issues.Get(ctx, r.Owner, r.Name, number)
	if err != nil {
		return nil, fmt.Errorf("unable to get issue %d for %s/%s: %w", number, r.Owner, r.Name, err)
	}

	return i, nil
}

func (r Repo) GetAllIssues(state string) (*[]github.Issue, error) {
	var allIssues []github.Issue

	if err := r.ListAllIssues(state, func(issues []*github.Issue, _ *github.Response) error {
		for index, i := range issues {
			if i == nil {
				clog.Log.Debugf("issues[%d] was nil, skipping", index)
				continue
			}

			n := i.GetNumber()
			if n == 0 {
				clog.Log.Debugf("issues[%d].Number was nil/0, skipping", index)
				continue
			}

			// return only issues and not pull requests.  All prs are issues, but not all issues are PRs
			if !i.IsPullRequest() {
				allIssues = append(allIssues, *i)
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to get all issues for %s/%s: %w", r.Owner, r.Name, err)
	}

	slices.SortFunc(allIssues, func(a, b github.Issue) int {
		return cmp.Compare(a.GetNumber(), b.GetNumber())
	})

	return &allIssues, nil
}
