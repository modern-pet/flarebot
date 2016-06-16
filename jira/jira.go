package jira

import (
	"net/http"
	"fmt"
	"errors"
	"bytes"
	"encoding/json"

	"io/ioutil"
)

type Ticket struct {
	Url string
	Key string
	ProjectId string
	ProjectKey string
	AssigneeEmail string
}

type User struct {
	Key string
	Name string
	Email string
}

type JiraService interface {
	GetUserByEmail(email string) (*User, error)
	GetTicketByKey(key string) (*Ticket, error)
	CreateTicket(priority int, topic string, assignee *User) (*Ticket, error)
}

// tuned for a single project
type JiraServer struct {
	Origin string
	Username string
	Password string
	ProjectID string
	IssueTypeID string
	PriorityIDs []string
}

func (server *JiraServer) DoRequest(path string, body *map[string]interface{}, expectArray bool) (map[string]interface{}, error) {
	fullURL := fmt.Sprintf("%s%s", server.Origin, path)

	var req *http.Request
	
	if body == nil {
		req, _ = http.NewRequest("GET", fullURL, nil)
	} else {
		jsonStr, _ := json.Marshal(body)
		req, _ = http.NewRequest("POST", fullURL, bytes.NewBuffer(jsonStr))
		req.Header.Add("Content-Type", "application/json")
	}

	req.SetBasicAuth(server.Username, server.Password)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Printf("got an error: %s\n", err)
		return nil, err
	}
	
	defer resp.Body.Close()

	responseBody, _ := ioutil.ReadAll(resp.Body)

	if expectArray {
		var response []map[string]interface{}
		json.Unmarshal(responseBody, &response)
		return response[0], nil
	} else {
		var response map[string]interface{}
		json.Unmarshal(responseBody, &response)
		return response, nil
	}
}


func (server *JiraServer) GetUserByEmail(email string) (*User, error) {
	response, err := server.DoRequest(fmt.Sprintf("/rest/api/2/user/search?username=%s", email), nil, true)

	if err != nil {
		return nil, err
	}
	
	return &User{
		Key: response["key"].(string),
		Name: response["name"].(string),
		Email: response["emailAddress"].(string),
	}, nil
}

func (server *JiraServer) GetTicketByKey(key string) (*Ticket, error) {
	response, err := server.DoRequest(fmt.Sprintf("/rest/api/2/issue/%s", key), nil, false)

	if err != nil {
		return nil, err
	}

	if response["fields"] == nil {
		return nil, errors.New("no such ticket")
	}
	
	var fields map[string]interface{} = response["fields"].(map[string]interface{})
	var project map[string]interface{} = fields["project"].(map[string]interface{})
	var assignee map[string]interface{} = fields["assignee"].(map[string]interface{})
	
	return &Ticket{
		Key: response["key"].(string),
		ProjectId: project["id"].(string),
		ProjectKey: project["key"].(string),
		AssigneeEmail: assignee["emailAddress"].(string),
	}, nil	
}

func (server *JiraServer) CreateTicket(priority int, topic string, assignee *User) (*Ticket, error) {
	// request JSON
	request := &map[string]interface{}{
		"fields": &map[string]interface{}{
			"project": &map[string]interface{}{
				"id": server.ProjectID,
			},
			"issuetype": &map[string]interface{}{
				"id": server.IssueTypeID,
			},
			"assignee": &map[string]interface{}{
				"name": assignee.Name,
			},
			"summary": topic,
			"priority": &map[string]interface{}{
				"id": server.PriorityIDs[priority],
			},
		},
	}

	url := "/rest/api/2/issue"
	response, _ := server.DoRequest(url, request, false)

	return &Ticket{
		Url: fmt.Sprintf("%s/issues/%s", server.Origin, response["key"]),
		Key: response["key"].(string),
	}, nil
}


