//go:build unit

package gits_test

import (
	"testing"

	"github.com/jenkins-x-plugins/jx-changelog/pkg/gits"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/stretchr/testify/assert"
)

func TestChangelogMarkdown(t *testing.T) {
	releaseSpec := &v1.ReleaseSpec{
		Version: "1",
		Commits: []v1.CommitSummary{
			{
				Message: "some commit 1\nfixes #123",
				SHA:     "123",
				Author: &v1.UserDetails{
					Name:  "James Strachan",
					Login: "jstrachan",
				},
			},
			{
				Message: "some commit 2\nfixes #345",
				SHA:     "456",
				Author: &v1.UserDetails{
					Name:  "James Rawlings",
					Login: "rawlingsj",
				},
			},
		},
	}
	gitInfo := &giturl.GitRepository{
		Host:         "github.com",
		Organisation: "jstrachan",
		Name:         "foo",
	}
	markdown, err := gits.GenerateMarkdown(releaseSpec, gitInfo, "", "", false, false)
	assert.Nil(t, err)
	//t.Log("Generated => " + markdown)

	expectedMarkdown := `## Changes in version 1

* some commit 1 ([jstrachan](https://github.com/jstrachan))
* some commit 2 ([rawlingsj](https://github.com/rawlingsj))
`
	assert.Equal(t, expectedMarkdown, markdown)
}

func TestChangelogMarkdownWithConventionalCommits(t *testing.T) {
	releaseSpec := &v1.ReleaseSpec{
		Version: "2",
		Commits: []v1.CommitSummary{
			{
				Message: "fix: some commit 1\nfixes #123",
				SHA:     "123",
				Author: &v1.UserDetails{
					Name:  "James Strachan",
					Login: "jstrachan",
				},
			},
			{
				Message: `feat: some commit 2
fixes #345
 loremm ipsum
BREAKING CHANGE: The git has fobbed!
`,
				SHA: "456",
				Author: &v1.UserDetails{
					Name:  "James Rawlings",
					Login: "rawlingsj",
				},
				IssueIDs: []string{"345"},
			},
			{
				Message: "feat(actual-feature-name)!: some commit 3\nfixes #456",
				SHA:     "567",
				Author: &v1.UserDetails{
					Name:  "James Rawlings",
					Login: "rawlingsj",
				},
				IssueIDs: []string{"456"},
			},
			{
				Message: "bad comment 4, see http://some.url/",
				SHA:     "678",
				Author: &v1.UserDetails{
					Name:  "James Rawlings",
					Login: "rawlingsj",
				},
			},
			{
				Message: "fresh eggs: bad comment 5",
				SHA:     "678",
				Author: &v1.UserDetails{
					Name:  "James Rawlings",
					Login: "rawlingsj",
				},
			},
			{
				Message:  "FOO-123: some other kind of commit\nFixes #345",
				IssueIDs: []string{"345"},
			},
		},
		Issues: []v1.IssueSummary{
			{
				ID:    "456",
				Title: "This needs to be fixed ASAP!",
				User: &v1.UserDetails{
					Name:  "James Strachan",
					Login: "jstrachan",
				},
				URL: "http://url-to-issue/456",
			},
			{
				ID:    "345",
				Title: "The shit has hit the fan!",
				User: &v1.UserDetails{
					Name:  "Mårten Svantesson",
					Login: "msvticket",
				},
				URL: "http://url-to-issue/345",
			},
		},
		PullRequests: []v1.IssueSummary{
			{
				ID:    "789",
				Title: "Upgrade of foo/bar to 1.2.3",
				Body: `Bumps foo/bar from 1.2.2 to 1.2.3.
-----
# bar

## Changes in version 1.2.3

### New Features

* The bar is open!
`,
				User: &v1.UserDetails{
					Name:  "Ankit",
					Login: "ankit",
				},
				URL: "http://url-to-pull/789",
			},
		},
	}
	gitInfo := &giturl.GitRepository{
		Host:         "github.com",
		Organisation: "jstrachan",
		Name:         "foo",
	}
	markdown, err := gits.GenerateMarkdown(releaseSpec, gitInfo, "-----", "-----", true, false)
	assert.Nil(t, err)
	//t.Log("Generated => " + markdown)

	expectedMarkdown := `## Changes in version 2

### FOO-123

* some other kind of commit [#345](http://url-to-issue/345) 

### BREAKING CHANGES

* The git has fobbed! ([rawlingsj](https://github.com/rawlingsj)) [#345](http://url-to-issue/345) 
* actual-feature-name: some commit 3 ([rawlingsj](https://github.com/rawlingsj)) [#456](http://url-to-issue/456) 

### New Features

* some commit 2 ([rawlingsj](https://github.com/rawlingsj)) [#345](http://url-to-issue/345) 

### Bug Fixes

* some commit 1 ([jstrachan](https://github.com/jstrachan))

### Other Changes

These commits did not use [Conventional Commits](https://conventionalcommits.org/) formatted messages:

* bad comment 4, see http://some.url/ ([rawlingsj](https://github.com/rawlingsj))
* fresh eggs: bad comment 5 ([rawlingsj](https://github.com/rawlingsj))

### Issues

* [#456](http://url-to-issue/456) This needs to be fixed ASAP! ([jstrachan](https://github.com/jstrachan))
* [#345](http://url-to-issue/345) The shit has hit the fan! ([msvticket](https://github.com/msvticket))

-----

# bar

## Changes in version 1.2.3

### New Features

* The bar is open!
`
	assert.Equal(t, expectedMarkdown, markdown)

}

// TODO: Add tests for JIRA as issue tracker
