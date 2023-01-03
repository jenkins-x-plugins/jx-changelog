//go:build integration

package create

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/releasereport"
	"github.com/jenkins-x/go-scm/scm"
	scmfake "github.com/jenkins-x/go-scm/scm/driver/fake"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	fakejx "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

func TestCreateChangelog(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err, "could not create temp dir")

	owner := "jstrachan"
	repo := "kubeconawesome"
	fullName := scm.Join(owner, repo)
	gitURL := "https://github.com/" + fullName

	scmClient, _ := scmfake.NewDefault()

	_, o := NewCmdChangelogCreate()

	g := o.Git()

	_, err = gitclient.CloneToDir(g, gitURL, tmpDir)
	require.NoError(t, err, "failed to clone %s", gitURL)

	o.JXClient = fakejx.NewSimpleClientset()
	o.Namespace = "jx"
	o.ScmFactory.Dir = tmpDir
	o.ScmFactory.ScmClient = scmClient
	o.ScmFactory.Owner = owner
	o.ScmFactory.Repository = repo
	o.BuildNumber = "1"
	o.Version = "2.0.1"
	o.GenerateReleaseYaml = true
	o.ExcludeRegexp = ""
	err = o.Run()
	require.NoError(t, err, "could not run changelog")

	f := filepath.Join(tmpDir, "charts", repo, "templates", "release.yaml")
	rel := AssertLoadReleaseYAML(t, f)

	commits := rel.Spec.Commits
	require.NotEmpty(t, commits, "no commits in file %s", f)
	for i := range commits {
		commit := commits[i]
		assert.NotEmpty(t, commit.SHA, "commit.SHA for commit %d in file %s", i, f)
		require.NotNil(t, commit.Author, "commit.Author for commit %d in file %s", i, f)
		assert.NotEmpty(t, commit.Author.Name, "commit.Author.Name for commit %d in file %s", i, f)
		assert.NotEmpty(t, commit.Author.Email, "commit.Author.Email for commit %d in file %s", i, f)

		t.Logf("commit %d is SHA %s user %s at %s\n", i, commit.SHA, commit.Author.Name, commit.Author.Email)
	}

	ctx := context.TODO()
	releases, _, err := scmClient.Releases.List(ctx, fullName, scm.ReleaseListOptions{})
	require.NoError(t, err, "failed to list releases on %s", fullName)
	require.Len(t, releases, 1, "should have one release for %s", fullName)
	release := releases[0]
	t.Logf("title: %s\n", release.Title)
	t.Logf("description: %s\n", release.Description)
	t.Logf("tag: %s\n", release.Tag)
}

// AssertLoadReleaseYAML asserts we can parse the release.yaml after stripping the helm conditional
func AssertLoadReleaseYAML(t *testing.T, f string) *v1.Release {
	require.FileExists(t, f, "should have created release file")

	rel := &v1.Release{}

	data, err := os.ReadFile(f)
	require.NoError(t, err, "failed to read file %s", f)

	releaseYAML := strings.TrimSpace(string(data))

	lines := strings.Split(releaseYAML, "\n")

	lastIdx := len(lines) - 1
	first := lines[0]
	last := lines[lastIdx]

	assert.True(t, strings.HasPrefix(first, "{{"), "release file %s first line should be conditional but was: %s", f, first)
	assert.True(t, strings.HasPrefix(last, "{{"), "release file %s last line should be conditional but was: %s", f, last)

	t.Logf("release first line conditional is the expected: %s\n", first)
	t.Logf("release last line conditional is the expected: %s\n", last)

	lines = lines[1:lastIdx]
	releaseYAML = strings.Join(lines, "\n")

	err = yaml.Unmarshal([]byte(releaseYAML), rel)
	require.NoError(t, err, "failed to parse file %s yaml: %s", f, releaseYAML)

	return rel
}

func TestCreateDependencyUpdates(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err, "could not create temp dir")

	owner := "jenkins-x-plugins"
	repo := "jx-gitops"
	fullName := scm.Join(owner, repo)
	gitURL := "https://github.com/" + fullName
	statusPath := "pkg/cmd/helmfile/report/testdata/releases.yaml"

	scmClient, _ := scmfake.NewDefault()

	_, o := NewCmdChangelogCreate()

	g := o.Git()

	_, err = gitclient.CloneToDir(g, gitURL, tmpDir)
	require.NoError(t, err, "failed to clone %s", gitURL)
	_, err = gitclient.CreateBranch(g, tmpDir)
	require.NoError(t, err, "failed to create branch")
	var currentReleases []*releasereport.NamespaceReleases
	absStatusPath := filepath.Join(tmpDir, statusPath)
	err = yamls.LoadFile(absStatusPath, &currentReleases)
	require.NoError(t, err, "failed to read %s", o.StatusPath)
	require.Greater(t, len(currentReleases), 1)
	releases := currentReleases[0].Releases
	require.NotEmpty(t, releases)
	testRel := releases[0]
	prevVersion := testRel.Version
	testRel.Version = "99.0.0"
	releases = currentReleases[2].Releases
	require.NotEmpty(t, releases)
	replaceRel := releases[0]
	oldName := replaceRel.ReleaseName
	replaceRel.ReleaseName = "the-new-name"
	err = yamls.SaveFile(currentReleases, absStatusPath)
	require.NoError(t, err, "failed to save status file")
	err = gitclient.Add(g, tmpDir, ".")
	require.NoError(t, err, "failed to add changes")
	err = gitclient.CommitIfChanges(g, tmpDir, "chore: upgrade")
	require.NoError(t, err, "failed to commit changes")
	_, err = g.Command(tmpDir, "tag", "v99.0.0")
	require.NoError(t, err, "failed to add tag")

	o.JXClient = fakejx.NewSimpleClientset()
	o.Namespace = "jx"
	o.ScmFactory.Dir = tmpDir
	o.ScmFactory.ScmClient = scmClient
	o.ScmFactory.Owner = owner
	o.ScmFactory.Repository = repo
	o.BuildNumber = "1"
	o.Version = "2.0.1"
	o.UpdateRelease = false
	o.StatusPath = statusPath
	o.OutputMarkdownFile = filepath.Join(tmpDir, "changelog.md")
	// o.LogLevel = "debug"
	err = o.Run()
	require.NoError(t, err, "could not run changelog")

	assert.FileExists(t, o.OutputMarkdownFile)
	markdown, err := os.ReadFile(o.OutputMarkdownFile)
	require.NoError(t, err, "failed to read markdown file")

	dependencyUpdates := fmt.Sprintf(`### Dependency Updates

| Component | New Version | Old Version |
| --------- | ----------- | ----------- |
| [%s](%s) | %s | %s |
| [%s](%s) | %s | %s |
| %s | %s | %s |`,
		testRel.ReleaseName, testRel.RepositoryURL, testRel.Version, prevVersion,
		replaceRel.ReleaseName, replaceRel.RepositoryURL, replaceRel.Version, "",
		oldName, "", replaceRel.Version)
	assert.Contains(t, string(markdown), dependencyUpdates)
}

func TestAddCommit(t *testing.T) {
	_, o := NewCmdChangelogCreate()

	release := v1.ReleaseSpec{
		Name:         SpecName,
		Commits:      []v1.CommitSummary{},
		Issues:       []v1.IssueSummary{},
		PullRequests: []v1.IssueSummary{},
	}
	exclude, _ := regexp.Compile(o.ExcludeRegexp)
	o.addCommit(&release, &object.Commit{Message: "release 1.0.0", Hash: plumbing.NewHash("123")}, nil, exclude)
	assert.Empty(t, release.Commits)
	exclude, _ = regexp.Compile(`^chore`)
	o.addCommit(&release, &object.Commit{Message: "chore: updated dependency", Hash: plumbing.NewHash("234")}, nil, exclude)
	assert.Empty(t, release.Commits)
	o.addCommit(&release, &object.Commit{Message: "feat: Cool new feature", Hash: plumbing.NewHash("234")}, nil, exclude)
	assert.Len(t, release.Commits, 1)
}
