package gh

import (
	"fmt"
	"strconv"
)

//
// TODO
// TODO - look for an actual SDK for github projects v2
// TODO
//

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
		Organization struct { // nolint: misspell
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
	// nolint: misspell
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

func (p *Project) HasItem(nodeID string) (*bool, error) {
	if p.ProjectDetails == nil {
		// todo should we do this automatically or error?
		return nil, fmt.Errorf("project details not loaded yet")
	}

	return nil, fmt.Errorf("not implemented")

	//couldn't get this to work
	/*q := `query=
	        query($project:ID!, $pr:ID!) {
	          projectV2(id: $project) {
	            items(first: 1, filterBy: {contentId: $pr}) {
	              nodes {
	                id
	              }
	            }
	          }
	        }
	    `

		fields := [][]string{
			{"-f", "pr=" + nodeID},
			{"-f", "project=" + p.ID},
		}

		// Execute the GraphQL query
		result, err := p.GraphQLQuery(q, fields)
		if err != nil {
			return nil, err
		}

		// Convert the result to a boolean
		exists := result != nil && *result == "true"
		return &exists, nil*/
}

func (p *Project) AddItem(nodeID string) (*string, error) {
	if p.ProjectDetails == nil {
		// todo should we do this automatically or error?
		return nil, fmt.Errorf("project details not loaded yet")
	}

	q := `query=
        mutation($project:ID!, $pr:ID!) {
          addProjectV2ItemById(input: {projectId: $project, contentId: $pr}) {
            item {
              id
            }
          }
        }
    `

	fields := [][]string{
		{"-f", "project=" + p.ID},
		{"-f", "pr=" + nodeID},
		{"--jq", ".data.addProjectV2ItemById.item.id"},
	}

	return p.GraphQLQuery(q, fields)
}

func (p *Project) SetItemStatus(itemId, status string) error {
	// should this be a method of ProjectItem? (to do this we'll need to figure out how to get all the fields and vlalues
	//TODO couldn't figure it out today)

	if p.ProjectDetails == nil {
		// todo should we do this automatically or error?
		return fmt.Errorf("project details not loaded yet")
	}

	q := `query=
					mutation (
                      $project:ID!, $item:ID!, 
                      $status_field:ID!, $status_value:String!,
					) {
					  set_status: updateProjectV2ItemFieldValue(input: {
						projectId: $project
						itemId: $item
						fieldId: $status_field
						value: { 
						  singleSelectOptionId: $status_value
						  }
					  }) {
						projectV2Item {
						  id
						  }
					  }
					}
				`

	fields := [][]string{
		{"-f", "project=" + p.ID},
		{"-f", "item=" + itemId},
		{"-f", "status_field=" + p.FieldIDs["Status"]},
		{"-f", "status_value=" + p.StatusIDs[status]},
	}

	_, err := p.GraphQLQuery(q, fields)
	return err
}

// for not we hard code the project fields we want (dueDate and type)
// TODO in the future we can make this configurable / get all of them
type ProjectItemsResult struct {
	Data struct {
		Organization struct {
			ProjectV2 struct {
				ID    string `json:"id"`
				Items struct {
					Nodes []struct {
						ID          string `json:"id"`
						Type        string `json:"type"`
						RequestType *struct {
							Text string `json:"text"`
						} `json:"requestType"`
						DueDate *struct {
							Date string `json:"date"`
						} `json:"dueDate"`
						Content struct {
							ID    string `json:"id"`
							Title string `json:"title"`
							URL   string `json:"url"`
						} `json:"content"`
					} `json:"nodes"`
				} `json:"items"`
			} `json:"projectV2"`
		} `json:"organization"`
	} `json:"data"`
}

type ProjectItem struct {
	ID          string
	Type        string
	Title       string
	URL         string
	RequestType string
	DueDate     string
	NodeID      string // actual pr/issue node id
}

// todo: allow configure the fields we want to get
func (p *Project) GetItems() ([]ProjectItem, error) {
	// nolint: misspell
	q := `query=
		query($org: String!, $number: Int!) {
			organization(login: $org) {
				projectV2(number: $number) {
					id
					items(first: 100) {
						nodes {
							id
							type
							requestType:fieldValueByName(name:"Type") {
								... on ProjectV2ItemFieldTextValue {
									text
								}
      						}
      						dueDate:fieldValueByName(name:"Due Date") {
        						... on ProjectV2ItemFieldDateValue {
             						date
           						}
      						}
							content {
								... on Issue {
									id
									title
									url
								}
								... on PullRequest {
									id
									title
									url
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

	var result ProjectItemsResult
	if err := p.GraphQLQueryUnmarshal(q, params, &result); err != nil {
		return nil, err
	}

	var items []ProjectItem
	for _, i := range result.Data.Organization.ProjectV2.Items.Nodes {
		item := ProjectItem{
			ID:     i.ID,
			Type:   i.Type,
			Title:  i.Content.Title,
			URL:    i.Content.URL,
			NodeID: i.Content.ID,
		}

		if i.RequestType != nil {
			item.RequestType = i.RequestType.Text
		}
		if i.DueDate != nil {
			item.DueDate = i.DueDate.Date
		}

		items = append(items, item)
	}

	return items, nil
}
