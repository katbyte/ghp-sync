package gh

import (
	"time"

	"github.com/shurcooL/githubv4"
)

type PullRequest struct {
	NodeID             string
	Author             string
	Number             int
	Title              string
	State              string
	ReviewDecision     string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	ClosedAt           time.Time
	Draft              bool
	Milestone          string
	TotalCommentsCount int
	TotalReviewCount   int

	Assignees                []string
	AssociatedLabels         map[string]bool
	AssociatedProjectNumbers map[int]bool
}

type pullRequestsQuery struct {
	Repository struct {
		PullRequests struct {
			Nodes []struct {
				Id                 string
				Number             int
				Title              string
				State              string
				ReviewDecision     string
				CreatedAt          time.Time
				UpdatedAt          time.Time
				ClosedAt           time.Time
				IsDraft            bool
				TotalCommentsCount int

				Assignees struct {
					Nodes []struct {
						Login string
					}
				} `graphql:"assignees(first: 10)"`

				Author struct {
					Login string
				}

				Labels struct {
					Nodes []struct {
						Name string
					}
				} `graphql:"labels(first: 100)"`

				Milestone struct {
					Title string
				}

				Reviews struct {
					Nodes []struct {
						Author struct {
							Login string
						}
					}
				} `graphql:"reviews(first: 100)"`

				ProjectItems struct {
					Nodes []struct {
						Project struct {
							Number int
						}
					}
				} `graphql:"projectItems(first: 10)"`
			}

			PageInfo struct {
				EndCursor   string
				HasNextPage bool
			}
		} `graphql:"pullRequests(first: 100, after: $cursor, states: $state, orderBy: {field: CREATED_AT, direction: DESC})"`
	} `graphql:"repository(owner: $owner, name: $repository)"`
}

func (r Repo) GetAllPullRequestsGQL(state githubv4.PullRequestState, reviewers []string) (*[]PullRequest, error) {
	client, ctx := r.NewGraphQLClient()

	allPRs := make([]PullRequest, 0)

	rev := make(map[string]struct{})
	if len(reviewers) != 0 {
		for _, reviewer := range reviewers {
			rev[reviewer] = struct{}{}
		}
	}

	query := pullRequestsQuery{}
	variables := map[string]any{
		"owner":      githubv4.String(r.Owner),
		"repository": githubv4.String(r.Name),
		"state":      []githubv4.PullRequestState{state},
		"cursor":     (*githubv4.String)(nil), // Default to nil / null, conditionally update this if there is pagination
	}

	for {
		if err := client.Query(ctx, &query, variables); err != nil {
			return nil, err
		}

		allPRs = append(allPRs, query.flatten(rev)...)

		if !query.Repository.PullRequests.PageInfo.HasNextPage {
			break
		}
		variables["cursor"] = githubv4.String(query.Repository.PullRequests.PageInfo.EndCursor)
	}

	return &allPRs, nil
}

func (q pullRequestsQuery) flatten(reviewers map[string]struct{}) []PullRequest {
	result := make([]PullRequest, 0)

	for _, pullRequest := range q.Repository.PullRequests.Nodes {
		pr := PullRequest{
			NodeID:                   pullRequest.Id,
			Author:                   pullRequest.Author.Login,
			Number:                   pullRequest.Number,
			Title:                    pullRequest.Title,
			State:                    pullRequest.State,
			ReviewDecision:           pullRequest.ReviewDecision,
			CreatedAt:                pullRequest.CreatedAt,
			UpdatedAt:                pullRequest.UpdatedAt,
			ClosedAt:                 pullRequest.ClosedAt,
			Draft:                    pullRequest.IsDraft,
			Milestone:                pullRequest.Milestone.Title,
			TotalCommentsCount:       pullRequest.TotalCommentsCount,
			AssociatedLabels:         make(map[string]bool),
			AssociatedProjectNumbers: make(map[int]bool),
		}

		for _, assignee := range pullRequest.Assignees.Nodes {
			pr.Assignees = append(pr.Assignees, assignee.Login)
		}

		for _, project := range pullRequest.ProjectItems.Nodes {
			pr.AssociatedProjectNumbers[project.Project.Number] = true
		}

		for _, label := range pullRequest.Labels.Nodes {
			pr.AssociatedLabels[label.Name] = true
		}

		// Only add review count if `reviewers` filter was provided
		// TODO: do we want separate fields, e.g. TotalReviews and TotalFilteredReviews?
		for _, review := range pullRequest.Reviews.Nodes {
			_, ok := reviewers[review.Author.Login]
			if ok {
				pr.TotalReviewCount++
			}
		}

		result = append(result, pr)
	}

	return result
}
