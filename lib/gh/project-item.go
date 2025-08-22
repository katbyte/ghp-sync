package gh

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func (p *Project) HasItem(nodeID string) (*bool, error) {
	if p.ProjectDetails == nil {
		// todo should we do this automatically or error?
		return nil, errors.New("project details not loaded yet")
	}

	return nil, errors.New("not implemented")

	// couldn't get this to work
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
		return nil, errors.New("project details not loaded yet")
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
	// should this be a method of ProjectItem? (to do this we'll need to figure out how to get all the fields and values
	// TODO couldn't figure it out today)

	if p.ProjectDetails == nil {
		// todo should we do this automatically or error?
		return errors.New("project details not loaded yet")
	}

	fields := []ProjectItemField{
		{Name: "number", FieldID: p.FieldIDs["Status"], Type: ItemValueTypeSingleSelect, Value: status},
	}

	return p.UpdateItem(itemId, fields)
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
						ID     string `json:"id"`
						Type   string `json:"type"`
						Status *struct {
							SingleSelectOptionID string `json:"singleSelectOptionId"`
						}
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
	Status      string
	NodeID      string // actual pr/issue node id
}

type ProjectItemFieldNameAndType struct {
	Name string        // The field name as it appears in GitHub's Project
	Type ItemValueType // The field type
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
							status:fieldValueByName(name:"Status") {
								... on ProjectV2ItemFieldSingleSelectValue {
									singleSelectOptionId		
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

	nodes := result.Data.Organization.ProjectV2.Items.Nodes
	items := make([]ProjectItem, len(nodes))
	for _, i := range nodes {
		item := ProjectItem{
			ID:     i.ID,
			Type:   i.Type,
			Title:  i.Content.Title,
			URL:    i.Content.URL,
			NodeID: i.Content.ID,
		}

		if i.Status != nil {
			item.Status = i.Status.SingleSelectOptionID
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

// create a project item field type NUMBER STRING
type ItemValueType int

const (
	ItemValueTypeText ItemValueType = iota
	ItemValueTypeNumber
	ItemValueTypeSingleSelect
	ItemValueTypeDate
)

// ProjectItemField represents a single field update for the project item.
// Type should be either "text" or "number".
type ProjectItemField struct {
	Name    string // A short name for this field (used in GraphQL alias, e.g. "set_key")
	FieldID string // The GraphQL ID of the field
	Type    ItemValueType
	Value   interface{}
}

// UpdateItem updates the fields of a project item by building a dynamic GraphQL mutation.
func (p *Project) UpdateItem(itemID string, fields []ProjectItemField) error {
	// We'll build the mutation parts dynamically; Always include project and item as variables
	varDefs := []string{"$project:ID!", "$item:ID!"}
	setCalls := []string{}
	params := [][]string{
		{"-f", "project=" + p.ID},
		{"-f", "item=" + itemID},
	}

	// For each field, we create a pair of variables: one for the fieldId, and one for the value
	// We'll name them based on the field's index to ensure uniqueness.
	for _, f := range fields {
		fieldAlias := f.Name
		if fieldAlias == "" {
			return errors.New("field name cannot be empty")
		}
		fieldIDVar := fmt.Sprintf("$%s_field", fieldAlias)
		fieldValueVar := fmt.Sprintf("$%s_value", fieldAlias)

		// Validate the field id it can never be empty
		if f.FieldID == "" {
			return fmt.Errorf("field ID for %s is empty", fieldAlias)
		}

		// Variable definitions based on Type
		varDefs = append(varDefs, fieldIDVar+":ID!")
		switch f.Type {
		case ItemValueTypeText:
			fallthrough
		case ItemValueTypeSingleSelect:
			varDefs = append(varDefs, fieldValueVar+":String!")
		case ItemValueTypeNumber:
			varDefs = append(varDefs, fieldValueVar+":Float!")
		case ItemValueTypeDate:
			varDefs = append(varDefs, fieldValueVar+":Date!")
		default:
			return fmt.Errorf("unsupported value type: %v for %s ", f.Type, fieldAlias)
		}

		// Add parameters for this field
		params = append(params, []string{"-f", fmt.Sprintf("%s_field=%s", fieldAlias, f.FieldID)})
		switch f.Type {
		case ItemValueTypeText:
			fallthrough
		case ItemValueTypeDate:
			fallthrough
		case ItemValueTypeSingleSelect:
			params = append(params, []string{"-f", fmt.Sprintf("%s_value=%v", fieldAlias, f.Value)})
		case ItemValueTypeNumber:
			// Use -F so the value is recognized as a JSON number
			params = append(params, []string{"-F", fmt.Sprintf("%s_value=%v", fieldAlias, f.Value)})
		}

		// Build the update call for the mutation
		var valuePart string
		switch f.Type {
		case ItemValueTypeText:
			valuePart = fmt.Sprintf("value: { text: %s }", fieldValueVar)
		case ItemValueTypeNumber:
			valuePart = fmt.Sprintf("value: { number: %s }", fieldValueVar)
		case ItemValueTypeDate:
			valuePart = fmt.Sprintf("value: { date: %s }", fieldValueVar)
		case ItemValueTypeSingleSelect:
			valuePart = fmt.Sprintf("value: { singleSelectOptionId: %s }", fieldValueVar)
		}

		setCalls = append(setCalls, fmt.Sprintf(`
  %s: updateProjectV2ItemFieldValue(input: {
    projectId: $project
    itemId: $item
    fieldId: %s
    %s
  }) {
    projectV2Item { id }
  }`, "set_"+fieldAlias, fieldIDVar, valuePart))
	}

	// Now assemble the full mutation
	mutation := fmt.Sprintf(`query=mutation(
  %s
) {
  %s
}`, strings.Join(varDefs, ", "), strings.Join(setCalls, "\n"))

	out, err := p.GraphQLQuery(mutation, params)
	if err != nil {
		return fmt.Errorf("error updating project item: %w: %s", err, *out)
	}

	return nil
}
