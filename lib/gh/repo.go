package gh

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/katbyte/ghp-sync/lib/pointer"
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
