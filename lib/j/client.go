package j

import (
	"context"

	jira "github.com/ctreminiom/go-atlassian/v2/jira/v3"
)

type Instance struct {
	URL   string
	User  string
	Token string
}

func NewInstance(url, user, token string) Instance {
	return Instance{
		URL:   url,
		User:  user,
		Token: token,
	}
}

func (i Instance) NewClient() (*jira.Client, context.Context, error) {
	ctx := context.Background()

	client, err := jira.New(nil, i.URL)
	if err != nil {
		return nil, nil, err
	}

	client.Auth.SetBasicAuth(i.User, i.Token)

	return client, ctx, nil
}
