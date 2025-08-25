package gh

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

func (t Token) GraphQLQueryUnmarshal(query string, params [][]string, data interface{}) error {
	out, err := t.GraphQLQuery(query, params)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(*out), data)
}

func (t Token) GraphQLQuery(query string, params [][]string) (*string, error) {
	args := []string{"api", "graphql", "-f", query}

	for _, p := range params {
		args = append(args, p[0])
		args = append(args, p[1])
	}

	// nolint: gosec // we are generating the args which _shouldn't_ contain anything a user could inject, and if it did users can be trusted
	ghc := exec.Command("gh", args...)
	if t.Token != nil {
		ghc.Env = []string{"GITHUB_TOKEN=" + *t.Token}
	}

	out, err := ghc.CombinedOutput()
	s := string(out)

	if err != nil {
		return &s, fmt.Errorf("graph ql query error: %w\n\n %s\n\n%s", err, query, out)
	}

	return &s, nil
}
