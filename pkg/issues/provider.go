package issues

import (
	"fmt"
	"time"

	"github.com/jenkins-x/go-scm/scm"
)

type IssueProvider interface {
	// GetIssue returns the issue of the given key
	GetIssue(key string) (*scm.Issue, error)

	// SearchIssues searches for issues (open by default)
	SearchIssues(query string) ([]*scm.Issue, error)

	// SearchIssuesClosedSince searches the issues closed since the given da
	SearchIssuesClosedSince(t time.Time) ([]*scm.Issue, error)

	// Creates a new issue in the current project
	CreateIssue(issue *scm.Issue) (*scm.Issue, error)

	// Creates a comment on the given issue
	CreateIssueComment(key string, comment string) error

	// IssueURL returns the URL of the given issue for this project
	IssueURL(key string) string

	// HomeURL returns the home URL of the issue tracker
	HomeURL() string
}

func CreateIssueProvider(kind string, serverURL, username, apiToken, project string, batchMode bool) (IssueProvider, error) {
	switch kind {
	case Jira:
		return CreateJiraIssueProvider(serverURL, username, apiToken, project, batchMode)
	default:
		return nil, fmt.Errorf("Unsupported issue provider kind: %s", kind)
	}
}

func ProviderAccessTokenURL(kind string, url string) string {
	switch kind {
	case Jira:
		// TODO handle on premise servers too by detecting the URL is at atlassian.com
		return "https://id.atlassian.com/manage/api-tokens"
	default:
		return ""
	}
}

// GetIssueProvider returns the kind of issue provider
func GetIssueProvider(tracker IssueProvider) string {
	_, ok := tracker.(*JiraService)
	if ok {
		return Jira
	}
	return Git
}
