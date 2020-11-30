package changelog_test

import (
	"io/ioutil"
	"testing"

	scmfake "github.com/jenkins-x/go-scm/scm/driver/fake"
	fakejx "github.com/jenkins-x/jx-api/v3/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"

	"github.com/jenkins-x-plugins/jx-changelog/pkg/cmd/changelog"
	"github.com/stretchr/testify/require"
)

func TestCommandChangelog(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	gitURL := "https://github.com/jstrachan/kubeconawesome"
	owner := "jstrachan"
	repo := "kubeconawesome"

	scmFake, _ := scmfake.NewDefault()

	_, o := changelog.NewCmdChangelogCreate()

	g := o.Git()

	_, err = gitclient.CloneToDir(g, gitURL, tmpDir)
	require.NoError(t, err, "failed to clone %s", gitURL)

	o.JXClient = fakejx.NewSimpleClientset()
	o.Namespace = "jx"
	o.ScmFactory.Dir = tmpDir
	o.ScmFactory.ScmClient = scmFake
	o.ScmFactory.Owner = owner
	o.ScmFactory.Repository = repo
	o.BuildNumber = "1"
	o.Version = "2.0.1"

	err = o.Run()
	require.NoError(t, err, "could not run changelog")

}
