package issues

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/pkg/errors"
)

type GitIssueProvider struct {
	GitProvider *scm.Client
	Owner       string
	Repository  string
	fullName    string
}

func CreateGitIssueProvider(scmClient *scm.Client, owner string, repository string) (IssueProvider, error) {
	if owner == "" {
		return nil, fmt.Errorf("No owner specified")
	}
	if repository == "" {
		return nil, fmt.Errorf("No owner specified")
	}
	fullName := scm.Join(owner, repository)
	return &GitIssueProvider{
		GitProvider: scmClient,
		Owner:       owner,
		Repository:  repository,
		fullName:    fullName,
	}, nil
}

func (i *GitIssueProvider) GetIssue(key string) (*scm.Issue, error) {
	ctx := context.Background()
	n, err := issueKeyToNumber(key)
	if err != nil {
		return nil, err
	}
	issue, _, err := i.GitProvider.Issues.Find(ctx, i.fullName, n)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find issue %d in repository %s", n, i.fullName)
	}
	return issue, nil
}

func (i *GitIssueProvider) SearchIssues(query string) ([]*scm.Issue, error) {
	ctx := context.Background()
	opts := scm.SearchOptions{
		Query: query,
	}
	searchIssues, _, err := i.GitProvider.Issues.Search(ctx, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to search issues with %v", opts)
	}
	var answer []*scm.Issue
	for _, si := range searchIssues {
		answer = append(answer, &si.Issue)
	}
	return answer, nil
}

func (i *GitIssueProvider) SearchIssuesClosedSince(t time.Time) ([]*scm.Issue, error) {
	// TODO
	//return i.GitProvider.SearchIssuesClosedSince(i.Owner, i.Repository, t)
	return nil, nil
}

func (i *GitIssueProvider) IssueURL(key string) string {
	return stringhelpers.UrlJoin(i.GitProvider.BaseURL.String(), i.fullName, "issues", key)
}

func issueKeyToNumber(key string) (int, error) {
	n, err := strconv.Atoi(key)
	if err != nil {
		return n, fmt.Errorf("Failed to convert issue key '%s' to number: %s", key, err)
	}
	return n, nil
}

func (i *GitIssueProvider) CreateIssue(issue *scm.Issue) (*scm.Issue, error) {
	return nil, errors.Errorf("TODO")
}

func (i *GitIssueProvider) CreateIssueComment(key string, comment string) error {
	ctx := context.Background()
	n, err := issueKeyToNumber(key)
	if err != nil {
		return err
	}
	ci := &scm.CommentInput{Body: comment}
	_, _, err = i.GitProvider.Issues.CreateComment(ctx, i.fullName, n, ci)
	if err != nil {
		return errors.Wrapf(err, "failed to add comment to issue %d on repository %s", n, i.fullName)
	}
	return nil
}

func (i *GitIssueProvider) HomeURL() string {
	return stringhelpers.UrlJoin(i.GitProvider.BaseURL.String(), i.Owner, i.Repository)
}
