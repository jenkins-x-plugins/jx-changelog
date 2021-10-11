package create_test

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghodss/yaml"

	"github.com/jenkins-x-plugins/jx-changelog/pkg/cmd/create"
	"github.com/jenkins-x/go-scm/scm"
	scmfake "github.com/jenkins-x/go-scm/scm/driver/fake"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	fakejx "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestCreateChangelog(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	owner := "jstrachan"
	repo := "kubeconawesome"
	fullName := scm.Join(owner, repo)
	gitURL := "https://github.com/" + fullName

	scmClient, _ := scmfake.NewDefault()

	_, o := create.NewCmdChangelogCreate()

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

	data, err := ioutil.ReadFile(f)
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
