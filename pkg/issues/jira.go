package issues

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
)

type JiraService struct {
	JiraClient *jira.Client
	ServerURL  string
	Project    string
}

func CreateJiraIssueProvider(serverURL, username, apiToken, project string, batchMode bool) (IssueProvider, error) {
	if serverURL == "" {
		return nil, fmt.Errorf("No JIRA server URL for server!")
	}
	var httpClient *http.Client
	if apiToken != "" {
		tp := jira.BasicAuthTransport{
			Username: username,
			Password: apiToken,
		}
		httpClient = tp.Client()
		if batchMode {
			log.Logger().Infof("Using JIRA server %s user name %s and an API token", serverURL, username)
		}
	} else {
		if batchMode {
			log.Logger().Warnf("No authentication found for JIRA server %s so using anonymous access", serverURL)
		}
	}
	jiraClient, _ := jira.NewClient(httpClient, serverURL)
	return &JiraService{
		JiraClient: jiraClient,
		ServerURL:  serverURL,
		Project:    project,
	}, nil
}

func (i *JiraService) GetIssue(key string) (*scm.Issue, error) {
	issue, _, err := i.JiraClient.Issue.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return i.jiraToGitIssue(issue), nil
}

func (i *JiraService) SearchIssues(query string) ([]*scm.Issue, error) {
	jql := "project = " + i.Project + " AND status NOT IN (Closed, Resolved)"
	if query != "" {
		jql += " AND text ~ " + query
	}
	var answer []*scm.Issue
	issues, _, err := i.JiraClient.Issue.Search(jql, nil)
	if err != nil {
		return answer, err
	}
	for _, issue := range issues {
		iss := issue
		answer = append(answer, i.jiraToGitIssue(&iss))
	}
	return answer, nil
}

func (i *JiraService) SearchIssuesClosedSince(_ time.Time) ([]*scm.Issue, error) {
	log.Logger().Warn("TODO SearchIssuesClosedSince() not yet implemented for JIRA")
	return nil, nil
}

func (i *JiraService) CreateIssue(issue *scm.Issue) (*scm.Issue, error) {
	project, _, err := i.JiraClient.Project.Get(i.Project)
	if err != nil {
		return nil, fmt.Errorf("Could not find project %s: %s", i.Project, err)
	}
	ji := i.gitToJiraIssue(issue)
	issueTypes := project.IssueTypes
	if len(issueTypes) > 0 {
		it := issueTypes[0]
		ji.Fields.Type.Name = it.Name
	}
	jira, resp, err := i.JiraClient.Issue.Create(ji)
	if err != nil {
		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(resp.Body)
		if err != nil {
			return nil, err
		}
		msg := buf.String()
		return nil, fmt.Errorf("Failed to create issue: %s due to: %s", msg, err)
	}
	return i.jiraToGitIssue(jira), nil
}

func (i *JiraService) CreateIssueComment(_ string, _ string) error {
	return fmt.Errorf("TODO")
}

func (i *JiraService) IssueURL(key string) string {
	return stringhelpers.UrlJoin(i.ServerURL, "browse", key)
}

func (i *JiraService) jiraToGitIssue(issue *jira.Issue) *scm.Issue {
	answer := &scm.Issue{}
	key := issue.Key
	// TODO
	//answer.Key = key
	answer.Link = i.IssueURL(key)
	fields := issue.Fields
	if fields != nil {
		answer.Title = fields.Summary
		answer.Body = fields.Description
		/// TODO
		//answer.Labels = gits.ToGitLabels(fields.Labels)
		// TODO
		//answer.ClosedAt = jiraTimeToTimeP(fields.Resolutiondate)
		user := jiraUserToGitUser(fields.Reporter)
		if user != nil {
			answer.Author = *user
		}
		assignee := jiraUserToGitUser(fields.Assignee)
		if assignee != nil {
			answer.Assignees = []scm.User{*assignee}
		}
	}
	return answer
}

func jiraUserToGitUser(user *jira.User) *scm.User {
	if user == nil {
		return nil
	}
	return &scm.User{
		Avatar: jiraAvatarUrl(user),
		Name:   user.Name,
		Login:  user.Key,
		Email:  user.EmailAddress,
	}
}
func jiraAvatarUrl(user *jira.User) string {
	answer := ""
	if user != nil {
		av := user.AvatarUrls
		answer = av.Four8X48
		if answer == "" {
			answer = av.Three2X32
		}
		if answer == "" {
			answer = av.Two4X24
		}
		if answer == "" {
			answer = av.One6X16
		}
	}
	return answer
}

func (i *JiraService) gitToJiraIssue(issue *scm.Issue) *jira.Issue {
	answer := &jira.Issue{
		Fields: &jira.IssueFields{
			Project: jira.Project{
				Key: i.Project,
			},
			Summary:     issue.Title,
			Description: issue.Body,
			Type: jira.IssueType{
				Name: "Bug",
			},
		},
	}
	return answer
}

func (i *JiraService) ServerName() string {
	return i.ServerURL
}

func (i *JiraService) HomeURL() string {
	return stringhelpers.UrlJoin(i.ServerURL, "browse", i.Project)
}
