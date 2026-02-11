package gh

import (
	"strconv"
)

type Project struct {
	Owner  string
	Number int
	Token

	*ProjectDetails
}

func NewProject(owner string, number int, token string) Project {
	p := Project{
		Owner:  owner,
		Number: number,
		Token: Token{
			Token: nil,
		},
	}

	if token != "" {
		p.Token.Token = &token
	}

	return p
}

type ProjectDetails struct {
	ID     string
	Fields []struct {
		ID      string
		Name    string
		Options []struct {
			ID   string
			Name string
		}
	}
	FieldIDs  map[string]string
	StatusIDs map[string]string
}

type ProjectDetailsResult struct {
	Data struct {
		Organization struct {
			ProjectV2 struct {
				ID     string `json:"id"`
				Fields struct {
					Nodes []struct {
						ID      string `json:"id"`
						Name    string `json:"name"`
						Options []struct {
							ID   string `json:"id"`
							Name string `json:"name"`
						} `json:"options"`
					} `json:"nodes"`
				} `json:"fields"`
			} `json:"projectV2"`
		} `json:"organization"`
	} `json:"data"`
}

func (p *Project) LoadDetails() error {
	q := `query=
        query($org: String!, $number: Int!) {
            organization(login: $org){
                projectV2(number: $number) {
                    id
                    fields(first:40) {
                        nodes {
                            ... on ProjectV2Field {
                                id
                                name
                            }
                            ... on ProjectV2SingleSelectField {
                                id
                                name
                                options {
                                    id
                                    name
                                }
                            }
                        }
                    }
                }
            }
        }
    `

	params := [][]string{
		{"-f", "org=" + p.Owner},
		{"-F", "number=" + strconv.Itoa(p.Number)},
	}

	var result ProjectDetailsResult
	if err := p.GraphQLQueryUnmarshal(q, params, &result); err != nil {
		return err
	}

	project := ProjectDetails{
		ID:        result.Data.Organization.ProjectV2.ID,
		FieldIDs:  map[string]string{},
		StatusIDs: map[string]string{},
	}

	for _, f := range result.Data.Organization.ProjectV2.Fields.Nodes {
		field := struct {
			ID      string
			Name    string
			Options []struct {
				ID   string
				Name string
			}
		}{
			ID:   f.ID,
			Name: f.Name,
		}

		if f.Name == "Status" {
			for _, s := range f.Options {
				field.Options = append(field.Options, struct {
					ID   string
					Name string
				}{
					ID:   s.ID,
					Name: s.Name,
				})
			}
		}

		project.Fields = append(project.Fields, field)
	}

	for _, f := range project.Fields {
		project.FieldIDs[f.Name] = f.ID

		if f.Name == "Status" {
			for _, s := range f.Options {
				project.StatusIDs[s.Name] = s.ID
			}
		}
	}

	p.ProjectDetails = &project
	return nil
}
