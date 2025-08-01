package create

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/imdario/mergo"
	"github.com/jenkins-x-plugins/jx-changelog/pkg/gits"
	"github.com/jenkins-x-plugins/jx-changelog/pkg/helmhelpers"
	"github.com/jenkins-x-plugins/jx-changelog/pkg/issues"
	"github.com/jenkins-x-plugins/jx-changelog/pkg/users"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/releasereport"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/variablefinders"
	"github.com/jenkins-x/go-scm/scm"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	jxc "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-helpers/v3/pkg/builds"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/activities"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/naming"

	"github.com/ghodss/yaml"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	jenkinsio "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Options contains the command line flags
type Options struct {
	options.BaseOptions

	ScmFactory    scmhelpers.Options
	GitClient     gitclient.Interface
	CommandRunner cmdrunner.CommandRunner
	JXClient      jxc.Interface

	Namespace                string
	BuildNumber              string
	PreviousRevision         string
	PreviousDate             string
	CurrentRevision          string
	TagPrefix                string
	TemplatesDir             string
	ReleaseYamlFile          string
	CrdYamlFile              string
	Version                  string
	Build                    string
	Header                   string
	HeaderFile               string
	Footer                   string
	FooterFile               string
	OutputMarkdownFile       string
	StatusPath               string
	ChangelogSeparator       string
	ChangelogOutputSeparator string
	IncludePRChangelog       bool
	OverwriteCRD             bool
	GenerateCRD              bool
	GenerateReleaseYaml      bool
	ConditionalRelease       bool
	UpdateRelease            bool
	NoReleaseInDev           bool
	IncludeMergeCommits      bool
	FailIfFindCommits        bool
	Draft                    bool
	Prerelease               bool
	State                    State
	ExcludeRegexp            string
	CompiledExcludeRegexp    *regexp.Regexp
}

type State struct {
	Tracker         issues.IssueProvider
	FoundIssueNames map[string]bool
	LoggedIssueKind bool
	Release         *v1.Release
}

const (
	ReleaseName = `{{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}`

	SpecName    = `{{ .Chart.Name }}`
	SpecVersion = `{{ .Chart.Version }}`

	ReleaseCrdYaml = `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: 2018-02-24T14:56:33Z
  name: releases.jenkins.io
  resourceVersion: "557150"
  selfLink: /apis/apiextensions.k8s.io/v1beta1/customresourcedefinitions/releases.jenkins.io
  uid: e77f4e08-1972-11e8-988e-42010a8401df
spec:
  group: jenkins.io
  names:
    kind: Release
    listKind: ReleaseList
    plural: releases
    shortNames:
    - rel
    singular: release
    categories:
    - all
  scope: Namespaced
  version: v1`
)

var (
	info = termcolor.ColorInfo

	AccessDescription = `

Jira API token is taken from the environment variable JIRA_API_TOKEN. Can be populated using the jx-boot-job-env-vars secret.

By default jx commands look for a file '~/.jx/gitAuth.yaml' to find the API tokens for Git servers. You can use 'jx create git token' to create a Git token.

Alternatively if you are running this command inside a CI server you can use environment variables to specify the username and API token.
e.g. define environment variables GIT_USERNAME and GIT_API_TOKEN
`

	cmdLong = templates.LongDesc(`
		Creates a Changelog for the latest tag

		This command will generate a Changelog as markdown for the git commit range given.
		If you are using GitHub it will also update the GitHub Release with the changelog. You can disable that by passing'--update-release=false'

		If you have just created a git tag this command will try default to the changes between the last tag and the previous one. You can always specify the exact Git references (tag/sha) directly via '--previous-rev' and '--rev'

		The changelog is generated by parsing the git commits. It will also detect any text like 'fixes #123' to link to issue fixes. You can also use Conventional Commits notation: https://conventionalcommits.org/ to get a nicer formatted changelog. e.g. using commits like 'fix:(my feature) this my fix' or 'feat:(cheese) something'

		This command also generates a Release Custom Resource Definition you can include in your helm chart to give metadata about the changelog of the application along with metadata about the release (git tag, url, commits, issues fixed etc). Including this metadata in a helm charts means we can do things like automatically comment on issues when they hit Staging or Production; or give detailed descriptions of what things have changed when using GitOps to update versions in an environment by referencing the fixed issues in the Pull Request.

		You can opt out of the release YAML generation via the '--generate-yaml=false' option

		To update the release notes on your git provider needs a git API token which is usually provided via the Tekton git authentication mechanism.

		Apart from using your git provider as the issue tracker there is also support for Jira. You then specify issues in commit messages with the issue key that looks like ABC-123. You can configure this in in similar ways as environments, see https://jenkins-x.io/v3/develop/environments/config/. An example configuration:

			issueProvider:
			  jira:
				serverUrl: https://example.atlassian.net
				userName: user@example.com
`) + AccessDescription

	cmdExample = templates.Examples(`
		# generate a changelog on the current source
		jx-changelog create

		# specify the version to use
		jx-changelog create --version 1.2.3

		# specify the version and a header template
		jx-changelog create --header-file docs/dev/changelog-header.md --version 1.2.3

`)

	GitHubIssueRegex = regexp.MustCompile(`\B#\d+\b`)
	JIRAIssueRegex   = regexp.MustCompile(`\b[A-Z][A-Z0-9_]+-\d+\b`)

	conditionalReleaseYAML = `{{- if and (.Capabilities.APIVersions.Has "jenkins.io/v1/Release") (hasKey .Values.jx "releaseCRD") (.Values.jx.releaseCRD)}}
%s 
{{- end }}
`
)

// NewCmdChangelogCreate creates the command and options
func NewCmdChangelogCreate() (*cobra.Command, *Options) {
	o := &Options{}
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Creates a changelog for a git tag",
		Aliases: []string{"changelog", "changes", "publish"},
		Long:    cmdLong,
		Example: cmdExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.ScmFactory.DiscoverFromGit = true

	cmd.Flags().StringVarP(&o.PreviousRevision, "previous-rev", "p", "", "the revision to start changelog from")
	cmd.Flags().StringVarP(&o.PreviousDate, "previous-date", "", "", "the date to start changelog from in format 'MonthName dayNumber year'")
	cmd.Flags().StringVarP(&o.CurrentRevision, "rev", "", "", "the revision to end changelog at")
	cmd.Flags().StringVarP(&o.TagPrefix, "tag-prefix", "", "", "prefix to filter on when searching for version tags")
	cmd.Flags().StringVarP(&o.TemplatesDir, "templates-dir", "t", "", "the directory containing the helm chart templates to generate the resources")
	cmd.Flags().StringVarP(&o.ReleaseYamlFile, "release-yaml-file", "", "release.yaml", "the name of the file to generate the Release YAML")
	cmd.Flags().StringVarP(&o.CrdYamlFile, "crd-yaml-file", "", "release-crd.yaml", "the name of the file to generate the Release CustomResourceDefinition YAML")
	cmd.Flags().StringVarP(&o.Version, "version", "v", "", "The version to release. Used to find the git tag to generate the changelog for and as title for the release")
	cmd.Flags().StringVarP(&o.Build, "build", "", "", "The Build number which is used to update the PipelineActivity. If not specified its defaulted from the '$BUILD_NUMBER' environment variable")
	cmd.Flags().StringVarP(&o.OutputMarkdownFile, "output-markdown", "", "", "Put the changelog output in this file")
	cmd.Flags().StringVarP(&o.StatusPath, "status-path", "", filepath.Join("docs", "releases.yaml"), "The path to the deployment status file used to calculate dependency updates.")
	cmd.Flags().StringVarP(&o.ChangelogSeparator, "changelog-separator", "", os.Getenv("CHANGELOG_SEPARATOR"), "the separator to use when splitting commit message from changelog in the pull request body. Default to ----- or if set the CHANGELOG_SEPARATOR environment variable")
	cmd.Flags().StringVarP(&o.ChangelogOutputSeparator, "changelog-output-separator", "", "-----", "the separator to use in changelog between changelogs from pull request bodies.")
	cmd.Flags().BoolVarP(&o.IncludePRChangelog, "include-changelog", "", true, "Should changelogs from pull requests be included.")
	cmd.Flags().BoolVarP(&o.OverwriteCRD, "overwrite", "o", false, "overwrites the Release CRD YAML file if it exists")
	cmd.Flags().BoolVarP(&o.GenerateCRD, "crd", "c", false, "Generate the CRD in the chart")
	cmd.Flags().BoolVarP(&o.GenerateReleaseYaml, "generate-yaml", "y", false, "Generate the Release YAML in the local helm chart")
	cmd.Flags().BoolVarP(&o.ConditionalRelease, "conditional-release", "", true, "Wrap the Release YAML in the helm Capabilities.APIVersions.Has if statement")
	cmd.Flags().BoolVarP(&o.UpdateRelease, "update-release", "", true, "Should we update the release on the Git repository with the changelog.")
	cmd.Flags().BoolVarP(&o.NoReleaseInDev, "no-dev-release", "", false, "Disables the generation of Release CRDs in the development namespace to track releases being performed")
	cmd.Flags().BoolVarP(&o.IncludeMergeCommits, "include-merge-commits", "", false, "Include merge commits when generating the changelog")
	cmd.Flags().BoolVarP(&o.FailIfFindCommits, "fail-if-no-commits", "", false, "Do we want to fail the build if we don't find any commits to generate the changelog")
	cmd.Flags().BoolVarP(&o.Draft, "draft", "", false, "The git provider release is marked as draft")
	cmd.Flags().BoolVarP(&o.Prerelease, "prerelease", "", false, "The git provider release is marked as a pre-release")
	defaultExcludeRegexp, ok := os.LookupEnv("CHANGELOG_EXCLUDE_REGEXP")
	if !ok {
		defaultExcludeRegexp = "^release "
	}
	cmd.Flags().StringVarP(&o.ExcludeRegexp, "exclude-regexp", "e", defaultExcludeRegexp, `Regexp for excluding commits. Can be set with environment variable CHANGELOG_EXCLUDE_REGEXP.`)

	cmd.Flags().StringVarP(&o.Header, "header", "", "", "The changelog header in markdown for the changelog. Can use go template expressions on the ReleaseSpec object: https://golang.org/pkg/text/template/")
	cmd.Flags().StringVarP(&o.HeaderFile, "header-file", "", "", "The file name of the changelog header in markdown for the changelog. Can use go template expressions on the ReleaseSpec object: https://golang.org/pkg/text/template/")
	cmd.Flags().StringVarP(&o.Footer, "footer", "", "", "The changelog footer in markdown for the changelog. Can use go template expressions on the ReleaseSpec object: https://golang.org/pkg/text/template/")
	cmd.Flags().StringVarP(&o.FooterFile, "footer-file", "", "", "The file name of the changelog footer in markdown for the changelog. Can use go template expressions on the ReleaseSpec object: https://golang.org/pkg/text/template/")

	o.ScmFactory.AddFlags(cmd)
	o.BaseOptions.AddBaseFlags(cmd)
	return cmd, o
}

func (o *Options) Validate() error {
	err := o.BaseOptions.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate base options: %w", err)
	}

	err = o.ScmFactory.Validate()
	if err != nil {
		return fmt.Errorf("failed to discover git repository: %w", err)
	}

	o.JXClient, o.Namespace, err = jxclient.LazyCreateJXClientAndNamespace(o.JXClient, o.Namespace)
	if err != nil {
		return fmt.Errorf("failed to create jx client: %w", err)
	}

	if o.ChangelogSeparator == "" {
		o.ChangelogSeparator = "-----"
	}
	if o.ExcludeRegexp != "" {
		o.CompiledExcludeRegexp, err = regexp.Compile(o.ExcludeRegexp)
		if err != nil {
			return fmt.Errorf("invalid regexp for option --exclude-regexp: %w", err)
		}
	}
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.QuietCommandRunner
	}

	return nil
}

func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate: %w", err)
	}

	// lets enable batch mode if we detect we are inside a pipeline
	if !o.BatchMode && builds.GetBuildNumber() != "" {
		log.Logger().Info("Using batch mode as inside a pipeline")
		o.BatchMode = true
	}

	dir := o.ScmFactory.Dir

	previousRev := o.PreviousRevision
	if previousRev == "" {
		previousDate := o.PreviousDate
		if previousDate != "" {
			previousRev, err = gits.GetRevisionBeforeDateText(o.Git(), dir, previousDate)
			if err != nil {
				return fmt.Errorf("failed to find commits before date %s: %w", previousDate, err)
			}
		}
	}
	ctx := context.Background()
	fullName := scm.Join(o.ScmFactory.Owner, o.ScmFactory.Repository)
	scmClient := o.ScmFactory.ScmClient

	if previousRev == "" {
		tagList, err := gits.NTags(o.Git(), dir, 11, o.TagPrefix)
		if err != nil {
			return fmt.Errorf("getting tags in %s: %w", dir, err)
		}
		if o.UpdateRelease && scmClient.Releases != nil {
			for n := 1; n < len(tagList); n++ {
				previousTag := tagList[n][1]
				// We ignore tags without releases so changelogs for failed release builds isn't skipped
				// TODO: Should we care about the status of the release?
				_, _, err = scmClient.Releases.FindByTag(ctx, fullName, previousTag)
				if err != nil {
					continue
				}
				previousRev, _, err = gits.GetCommitForTagSha(o.Git(), dir, tagList[n][0], previousTag)
				if err != nil {
					return err
				}
				break
			}
		}
		if previousRev == "" {
			if len(tagList) > 1 {
				// If no release was found use the first tag before current
				previousRev, _, err = gits.GetCommitForTagSha(o.Git(), dir, tagList[1][0], tagList[1][1])
				if err != nil {
					return err
				}
			} else {
				// let's assume we are the first release
				previousRev, err = gits.GetFirstCommitSha(o.Git(), dir)
				if err != nil {
					return fmt.Errorf("failed to find first commit after we found no previous releaes: %w", err)
				}
				if previousRev == "" {
					log.Logger().Info("no previous commit version found so change diff unavailable")
					return nil
				}
			}
		}
	}
	currentRev, tagName, err := gits.GetCommitPointedToByLatestTag(o.Git(), dir, o.TagPrefix)
	if err != nil {
		return err
	}
	if o.CurrentRevision != "" {
		currentRev = o.CurrentRevision
	}
	prefix := "v"
	if o.TagPrefix != "" {
		prefix = o.TagPrefix
	}
	version := o.Version
	if version != "" && version != tagName && prefix+version != tagName {
		log.Logger().Warnf("version %s does not match the latest tag %s. Will tag head of default branch with version", info(version), info(tagName))
		tagName = version
	}
	if version == "" {
		version = tagName
	}

	templatesDir := o.TemplatesDir
	if templatesDir == "" {
		chartFile, err := helmhelpers.FindChart(dir)
		if err != nil {
			return fmt.Errorf("could not find helm chart: %w", err)
		}
		if chartFile == "" {
			log.Logger().Infof("no chart directory found in %s", dir)
			templatesDir = ""
		} else {
			path, _ := filepath.Split(chartFile)
			if path == "" {
				log.Logger().Infof("no chart directory found in %s", dir)
				templatesDir = ""
			} else {
				templatesDir = filepath.Join(path, "templates")
			}
		}
	}
	if templatesDir != "" {
		err = os.MkdirAll(templatesDir, files.DefaultDirWritePermissions)
		if err != nil {
			return fmt.Errorf("failed to create the templates directory %s: %w", templatesDir, err)
		}
	}

	log.Logger().Infof("Generating change log from git ref %s => %s", info(previousRev), info(currentRev))

	gitDir, gitConfDir, err := gitclient.FindGitConfigDir(dir)
	if err != nil {
		return err
	}
	if gitDir == "" || gitConfDir == "" {
		log.Logger().Warnf("No git directory could be found from dir %s", dir)
		return nil
	}

	gitInfo := o.ScmFactory.GitURL
	if gitInfo == nil {
		gitInfo, err = giturl.ParseGitURL(o.ScmFactory.SourceURL)
		if err != nil {
			return fmt.Errorf("failed to parse git URL %s: %w", o.ScmFactory.SourceURL, err)
		}
	}

	tracker, err := o.CreateIssueProvider()
	if err != nil {
		return err
	}
	o.State.Tracker = tracker

	o.State.FoundIssueNames = map[string]bool{}

	commits, err := FetchCommits(gitDir, previousRev, currentRev)
	if err != nil {
		if o.FailIfFindCommits {
			return err
		}
		log.Logger().Warnf("failed to find git commits between revision %s and %s due to: %s", previousRev, currentRev, err.Error())
	} else if log.Logger().Logger.IsLevelEnabled(logrus.DebugLevel) {
		log.Logger().Debugf("Found commits:")
		for k := range *commits {
			commit := (*commits)[k]
			log.Logger().Debugf("  commit %s", commit.Hash)
			log.Logger().Debugf("  Author: %s <%s>", commit.Author.Name, commit.Author.Email)
			log.Logger().Debugf("  Date: %s", commit.Committer.When.Format(time.ANSIC))
			log.Logger().Debugf("      %s\n\n\n", commit.Message)
		}
	}

	version = strings.TrimPrefix(version, prefix)
	specVersion := version
	if specVersion == "" {
		specVersion = SpecVersion
	}

	release := &v1.Release{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Release",
			APIVersion: jenkinsio.GroupAndVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ReleaseName,
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
			// ResourceVersion:   "1",
			DeletionTimestamp: &metav1.Time{},
		},
		Spec: v1.ReleaseSpec{
			Name:          SpecName,
			Version:       specVersion,
			GitOwner:      gitInfo.Organisation,
			GitRepository: gitInfo.Name,
			GitHTTPURL:    gitInfo.HttpsURL(),
			GitCloneURL:   gitInfo.CloneURL,
			Commits:       []v1.CommitSummary{},
			Issues:        []v1.IssueSummary{},
			PullRequests:  []v1.IssueSummary{},
		},
	}

	resolver := users.GitUserResolver{
		GitProvider: scmClient,
	}
	if commits != nil {
		for k := range *commits {
			c := (*commits)[k]
			o.addCommit(&release.Spec, &c, &resolver, o.CompiledExcludeRegexp)
		}
	}

	release.Spec.DependencyUpdates, err = o.getDependencyUpdates(previousRev)
	if err != nil {
		log.Logger().Warnf("failed to get dependency updates: %v", err)
	}

	// let's try to update the release
	markdown, err := gits.GenerateMarkdown(&release.Spec, gitInfo, o.ChangelogSeparator, o.ChangelogOutputSeparator, o.IncludePRChangelog, o.IncludeMergeCommits)
	if err != nil {
		return err
	}
	header, err := o.getTemplateResult(&release.Spec, "header", o.Header, o.HeaderFile)
	if err != nil {
		return err
	}
	footer, err := o.getTemplateResult(&release.Spec, "footer", o.Footer, o.FooterFile)
	if err != nil {
		return err
	}
	markdown = header + markdown + footer
	markdownOutputted := false
	log.Logger().Debugf("Generated release notes:\n\n%s\n", markdown)

	if version != "" && o.UpdateRelease {
		releaseInfo := &scm.ReleaseInput{
			Title:       version,
			Tag:         tagName,
			Description: markdown,
			Draft:       o.Draft,
			Prerelease:  o.Prerelease,
		}

		// let's try to find a release for the tag
		if scmClient.Releases == nil {
			log.Logger().Warnf("scm provider does not support Releases so cannot find releases")
		} else {
			rel, _, err := scmClient.Releases.FindByTag(ctx, fullName, tagName)

			if isReleaseNotFound(err, o.ScmFactory.GitKind) {
				err = nil
				rel = nil
			}
			if err != nil {
				return fmt.Errorf("failed to query release on repo %s for tag %s: %w", fullName, tagName, err)
			}

			if rel == nil {
				rel, _, err = scmClient.Releases.Create(ctx, fullName, releaseInfo)
				if err != nil {
					log.Logger().Warnf("Failed to create the release for %s: %s", fullName, err)
					return nil
				}
			} else {
				if rel.ID != 0 {
					rel, _, err = scmClient.Releases.Update(ctx, fullName, rel.ID, releaseInfo)
				} else {
					rel, _, err = scmClient.Releases.UpdateByTag(ctx, fullName, rel.Tag, releaseInfo)
				}
				if err != nil {
					id := -1
					if rel != nil {
						id = rel.ID
					}
					log.Logger().Warnf("Failed to update the release for %s number: %d: %s", fullName, id, err)
					return nil
				}
			}

			url := ""
			if rel != nil {
				url = rel.Link
			}
			if url == "" {
				url = stringhelpers.UrlJoin(gitInfo.HttpsURL(), "releases/tag", tagName)
			}
			release.Spec.ReleaseNotesURL = url
			log.Logger().Infof("updated the release information at %s", info(url))
			log.Logger().Debugf("added description: %s", markdown)
			markdownOutputted = true
		}
	}

	if o.OutputMarkdownFile != "" {
		err := os.WriteFile(o.OutputMarkdownFile, []byte(markdown), files.DefaultFileWritePermissions)
		if err != nil {
			return err
		}
		log.Logger().Infof("\nGenerated Changelog: %s", info(o.OutputMarkdownFile))
		markdownOutputted = true
	}
	if !markdownOutputted {
		log.Logger().Infof("\nGenerated Changelog:")
		log.Logger().Infof("%s\n", markdown)
	}

	o.State.Release = release
	// now lets marshal the release YAML
	data, err := yaml.Marshal(release)
	if o.ConditionalRelease {
		data = []byte(fmt.Sprintf(conditionalReleaseYAML, string(data)))
	}

	if err != nil {
		return fmt.Errorf("failed to unmarshal Release: %w", err)
	}
	if data == nil {
		return fmt.Errorf("could not marshal release to yaml")
	}

	if templatesDir != "" {
		releaseFile := filepath.Join(templatesDir, o.ReleaseYamlFile)
		crdFile := filepath.Join(templatesDir, o.CrdYamlFile)
		if o.GenerateReleaseYaml {
			err = os.WriteFile(releaseFile, data, files.DefaultFileWritePermissions)
			if err != nil {
				return fmt.Errorf("failed to save Release YAML file %s: %w", releaseFile, err)
			}
			log.Logger().Infof("generated: %s", info(releaseFile))
		}
		if o.GenerateCRD {
			exists, err := files.FileExists(crdFile)
			if err != nil {
				return fmt.Errorf("failed to check for CRD YAML file %s: %w", crdFile, err)
			}
			if o.OverwriteCRD || !exists {
				err = os.WriteFile(crdFile, []byte(ReleaseCrdYaml), files.DefaultFileWritePermissions)
				if err != nil {
					return fmt.Errorf("failed to save Release CRD YAML file %s: %w", crdFile, err)
				}
				log.Logger().Infof("generated: %s", info(crdFile))

				err = gitclient.Add(o.Git(), templatesDir)
				if err != nil {
					return fmt.Errorf("failed to git add in dir %s: %w", templatesDir, err)
				}
			}
		}
	}
	releaseNotesURL := release.Spec.ReleaseNotesURL

	// let's modify the PipelineActivity
	err = o.updatePipelineActivity(func(pa *v1.PipelineActivity) (bool, error) {
		updated := false
		ps := &pa.Spec

		doUpdate := func(oldValue, newValue string) string {
			if newValue == "" || newValue == oldValue {
				return oldValue
			}
			updated = true
			return newValue
		}

		commits := release.Spec.Commits
		if len(commits) > 0 {
			lastCommit := commits[len(commits)-1]
			ps.LastCommitSHA = doUpdate(ps.LastCommitSHA, lastCommit.SHA)
			ps.LastCommitMessage = doUpdate(ps.LastCommitMessage, lastCommit.Message)
			ps.LastCommitURL = doUpdate(ps.LastCommitURL, lastCommit.URL)
		}
		ps.ReleaseNotesURL = doUpdate(ps.ReleaseNotesURL, releaseNotesURL)
		ps.Version = doUpdate(ps.Version, version)
		return updated, nil
	})
	if err != nil {
		return fmt.Errorf("failed to update PipelineActivity: %w", err)
	}
	return nil
}

// FindIssueTracker finds the issue tracker from the settings in current repo as well as sourcerepositories and
// requirements from cluster repo
func FindIssueTracker(g gitclient.Interface, jxClient jxc.Interface, ns, dir, owner, repo string) (*jxcore.IssueTracker, error) {
	// now lets merge the local requirements with the dev environment so that we can locally override things
	// while inheriting common stuff
	settings, clusterDir, err := variablefinders.GetSettings(g, jxClient, ns, dir, owner, repo)
	if err != nil {
		return nil, err
	}

	requirementsConfig, _, err := jxcore.LoadRequirementsConfig(clusterDir, false)
	var reqIssueTracker *jxcore.IssueTracker
	if err != nil {
		return nil, fmt.Errorf("cannot load requirements config file: %w", err)
	}
	if requirementsConfig != nil && !requirementsConfig.Spec.IsEmpty() {
		reqIssueTracker = requirementsConfig.Spec.Cluster.IssueTracker
	}

	issueTracker := settings.Spec.IssueTracker
	if reqIssueTracker != nil {
		if issueTracker != nil {
			err = mergo.Merge(reqIssueTracker, issueTracker, mergo.WithOverride)
			if err != nil {
				return nil, fmt.Errorf("error merging requirements.spec.cluster Destination from settings: %w", err)
			}
		}
		return reqIssueTracker, nil
	}
	return issueTracker, nil
}

func (o *Options) updatePipelineActivity(fn func(activity *v1.PipelineActivity) (bool, error)) error {
	if o.BuildNumber == "" {
		o.BuildNumber = os.Getenv("BUILD_NUMBER")
		if o.BuildNumber == "" {
			o.BuildNumber = os.Getenv("BUILD_ID")
		}
	}
	pipeline := fmt.Sprintf("%s/%s/%s", o.ScmFactory.Owner, o.ScmFactory.Repository, o.ScmFactory.Branch)

	ctx := context.Background()
	build := o.BuildNumber
	if pipeline != "" && build != "" {
		ns := o.Namespace
		name := naming.ToValidName(pipeline + "-" + build)

		jxClient := o.JXClient

		// lets see if we can update the pipeline
		acts := jxClient.JenkinsV1().PipelineActivities(ns)
		key := &activities.PromoteStepActivityKey{
			PipelineActivityKey: activities.PipelineActivityKey{
				Name:     name,
				Pipeline: pipeline,
				Build:    build,
				GitInfo: &giturl.GitRepository{
					Name:         o.ScmFactory.Repository,
					Organisation: o.ScmFactory.Owner,
				},
			},
		}

		var lastErr error
		for i := 0; i < 3; i++ {
			a, _, err := key.GetOrCreate(o.JXClient, o.Namespace)
			if err != nil {
				return fmt.Errorf("failed to get PipelineActivity: %w", err)
			}

			updated, err := fn(a)
			if err != nil {
				return fmt.Errorf("failed to update PipelineActivit %s: %w", name, err)
			}
			if !updated {
				return nil
			}
			a, err = acts.Update(ctx, a, metav1.UpdateOptions{})
			if err != nil {
				lastErr = err
			} else {
				log.Logger().Infof("Updated PipelineActivity %s which has status %s", name, string(a.Spec.Status))
				return nil
			}
		}
		if lastErr != nil {
			log.Logger().Warnf("failed to update  PipelineActivity %s due to %s", name, lastErr.Error())
		}
	} else {
		log.Logger().Warnf("No $BUILD_NUMBER so cannot update PipelineActivities with the details from the changelog")
	}
	return nil
}

// CreateIssueProvider creates the issue provider
func (o *Options) CreateIssueProvider() (issues.IssueProvider, error) {
	issueTracker, _ := FindIssueTracker(o.Git(), o.JXClient, "", o.ScmFactory.Dir, o.ScmFactory.Owner, o.ScmFactory.Repository)
	if issueTracker != nil && issueTracker.Jira != nil {
		j := issueTracker.Jira
		jiraAPIToken := os.Getenv("JIRA_API_TOKEN")
		if jiraAPIToken != "" {
			return issues.CreateJiraIssueProvider(j.ServerURL, j.Username, jiraAPIToken, j.Project, true)
		}
		log.Logger().Warnf("Environment variable JIRA_API_TOKEN can't be found so connection to JIRA can't be made")

	}
	log.Logger().Infof("Can't find any issue tracker setting; defaulting to git provider: %s",
		o.ScmFactory.ScmClient.Driver.String())
	return issues.CreateGitIssueProvider(o.ScmFactory.ScmClient, o.ScmFactory.Owner, o.ScmFactory.Repository)
}

func (o *Options) Git() gitclient.Interface {
	if o.GitClient == nil {
		o.GitClient = cli.NewCLIClient("", o.CommandRunner)
	}
	return o.GitClient
}

func (o *Options) addCommit(spec *v1.ReleaseSpec, commit *object.Commit, resolver *users.GitUserResolver, excludeRegexp *regexp.Regexp) {
	if (!(o.IncludeMergeCommits || o.IncludePRChangelog || len(commit.ParentHashes) <= 1)) ||
		(excludeRegexp != nil && excludeRegexp.MatchString(commit.Message)) {
		return
	}
	url := ""
	branch := "master"

	var author, committer *v1.UserDetails
	var err error
	sha := commit.Hash.String()
	if commit.Author.Email != "" && commit.Author.Name != "" {
		author, err = resolver.GitSignatureAsUser(&commit.Author)
		if err != nil {
			log.Logger().Warnf("failed to enrich commit with issues, error getting git signature for git author %s: %v", commit.Author, err)
		}
	}
	if commit.Committer.Email != "" && commit.Committer.Name != "" {
		committer, err = resolver.GitSignatureAsUser(&commit.Committer)
		if err != nil {
			log.Logger().Warnf("failed to enrich commit with issues, error getting git signature for git committer %s: %v", commit.Committer, err)
		}
	}
	commitSummary := v1.CommitSummary{
		Message:   commit.Message,
		URL:       url,
		SHA:       sha,
		Author:    author,
		Branch:    branch,
		Committer: committer,
	}

	o.addIssuesAndPullRequests(spec, &commitSummary, commit)
	if o.IncludeMergeCommits || len(commit.ParentHashes) <= 1 {
		spec.Commits = append(spec.Commits, commitSummary)
	}
}

func (o *Options) addIssuesAndPullRequests(spec *v1.ReleaseSpec, commit *v1.CommitSummary, rawCommit *object.Commit) {
	tracker := o.State.Tracker

	issueKind := issues.GetIssueProvider(tracker)
	if !o.State.LoggedIssueKind {
		o.State.LoggedIssueKind = true
		log.Logger().Infof("Finding issues in commit messages using %s format", issueKind)
	}
	if issueKind == issues.Jira {
		o.addIssuesAndPullRequestsWithPattern(spec, commit, JIRAIssueRegex, rawCommit.Message, tracker)
	}

	o.addIssuesAndPullRequestsWithPattern(spec, commit, GitHubIssueRegex, rawCommit.Message, tracker)
}

func (o *Options) addIssuesAndPullRequestsWithPattern(spec *v1.ReleaseSpec, commit *v1.CommitSummary, regex *regexp.Regexp, message string, tracker issues.IssueProvider) {
	matches := regex.FindAllString(message, -1)

	resolver := users.GitUserResolver{
		GitProvider: o.ScmFactory.ScmClient,
	}
	for _, result := range matches {
		result = strings.TrimPrefix(result, "#")
		if issueExists, ok := o.State.FoundIssueNames[result]; !ok {
			o.State.FoundIssueNames[result] = false
			issue, err := tracker.GetIssue(result)
			if err != nil {
				log.Logger().Warnf("Failed to lookup issue %s in issue tracker %s due to %s", result, tracker.HomeURL(), err)
				continue
			}
			if issue == nil {
				log.Logger().Warnf("Failed to find issue %s for repository %s", result, tracker.HomeURL())
				continue
			}
			o.State.FoundIssueNames[result] = true
			commit.IssueIDs = append(commit.IssueIDs, result)

			var user *v1.UserDetails
			if issues.GetIssueProvider(tracker) == issues.Git {
				user, err = resolver.Resolve(&issue.Author)
				if err != nil {
					log.Logger().Warnf("Failed to resolve user %v for issue %s repository %s", issue.Author, result, tracker.HomeURL())
				}
			} else {
				auth := &issue.Author
				user = &v1.UserDetails{
					Login:     auth.Login,
					Name:      auth.Name,
					Email:     auth.Email,
					URL:       auth.Link,
					AvatarURL: auth.Avatar,
				}
			}

			var assignees []v1.UserDetails
			if issue.Assignees == nil {
				log.Logger().Warnf("Failed to find assignees for issue %s repository %s", result, tracker.HomeURL())
			} else {
				u, err := resolver.GitUserSliceAsUserDetailsSlice(issue.Assignees)
				if err != nil {
					log.Logger().Warnf("Failed to resolve Assignees %v for issue %s repository %s", issue.Assignees, result, tracker.HomeURL())
				}
				assignees = u
			}

			labels := toV1Labels(issue.Labels)
			issueSummary := v1.IssueSummary{
				ID:                result,
				URL:               issue.Link,
				Title:             issue.Title,
				Body:              issue.Body,
				User:              user,
				CreationTimestamp: kube.ToMetaTime(&issue.Created),
				Assignees:         assignees,
				Labels:            labels,
			}
			state := issue.State
			if state != "" {
				issueSummary.State = state
			}
			if issue.PullRequest != nil {
				spec.PullRequests = append(spec.PullRequests, issueSummary)
			} else {
				spec.Issues = append(spec.Issues, issueSummary)
			}
		} else if issueExists {
			commit.IssueIDs = stringhelpers.EnsureStringArrayContains(commit.IssueIDs, result)
		}
	}
}

// toV1Labels converts git labels to IssueLabel
func toV1Labels(labels []string) []v1.IssueLabel {
	var answer []v1.IssueLabel
	for _, label := range labels {
		answer = append(answer, v1.IssueLabel{
			Name: label,
		})
	}
	return answer
}

func (o *Options) getTemplateResult(releaseSpec *v1.ReleaseSpec, templateName, templateText, templateFile string) (string, error) {
	if templateText == "" {
		if templateFile == "" {
			return "", nil
		}
		data, err := os.ReadFile(templateFile)
		if err != nil {
			return "", err
		}
		templateText = string(data)
	}
	if templateText == "" {
		return "", nil
	}
	tmpl, err := template.New(templateName).Parse(templateText)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	writer := bufio.NewWriter(&buffer)
	err = tmpl.Execute(writer, releaseSpec)
	flushErr := writer.Flush()
	if err == nil {
		err = flushErr
	}
	return buffer.String(), err
}

func (o *Options) getDependencyUpdates(previousRev string) ([]v1.DependencyUpdate, error) {
	dir := o.ScmFactory.Dir
	absStatusPath := filepath.Join(dir, o.StatusPath)
	releasesExists, err := files.FileExists(absStatusPath)
	if err != nil {
		log.Logger().Debugf("fail to check if %s exists", absStatusPath)
		return nil, nil
	}
	if !releasesExists {
		log.Logger().Debugf("file %s doesn't exists", absStatusPath)
		return nil, nil
	}
	previousReleasesBlob, err := o.Git().Command(dir, "cat-file", "blob", previousRev+":"+o.StatusPath)
	if err != nil {
		return nil, fmt.Errorf("fail to check if %s exists for %s: %w", o.StatusPath, previousRev, err)
	}
	var previousReleases []*releasereport.NamespaceReleases
	err = yaml.Unmarshal([]byte(previousReleasesBlob), &previousReleases)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal previous releases %s: %w", previousRev, err)
	}

	var currentReleases []*releasereport.NamespaceReleases
	err = yamls.LoadFile(absStatusPath, &currentReleases)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", o.StatusPath, err)
	}

	previousReleasesMap := makeReleaseMap(&previousReleases)
	updates := make([]v1.DependencyUpdate, 0)

	for _, nsr := range currentReleases {
		prevReleases, nsexisted := previousReleasesMap[nsr.Namespace]
		if !nsexisted {
			prevReleases = make(map[string]string)
		}
		for _, release := range nsr.Releases {
			prevRel, relexisted := prevReleases[release.ReleaseName]
			if relexisted {
				delete(prevReleases, release.ReleaseName)
			}
			if prevRel != release.Version {
				url := release.RepositoryURL
				if url == "" {
					url = release.ApplicationURL
				}
				updates = append(updates, v1.DependencyUpdate{
					DependencyUpdateDetails: v1.DependencyUpdateDetails{
						Component:   release.ReleaseName,
						URL:         url,
						FromVersion: prevRel,
						ToVersion:   release.Version,
					},
				})
			}
		}
	}

	for _, nsr := range previousReleasesMap {
		for name, release := range nsr {
			updates = append(updates, v1.DependencyUpdate{
				DependencyUpdateDetails: v1.DependencyUpdateDetails{
					Component:   name,
					FromVersion: release,
				},
			})
		}
	}

	return updates, nil
}

func makeReleaseMap(namespaceReleases *[]*releasereport.NamespaceReleases) map[string]map[string]string {
	res := make(map[string]map[string]string)
	for _, nsr := range *namespaceReleases {
		res[nsr.Namespace] = make(map[string]string)
		for _, release := range nsr.Releases {
			res[nsr.Namespace][release.ReleaseName] = release.Version
		}
	}
	return res
}

func isReleaseNotFound(err error, gitKind string) bool {
	switch gitKind {
	case "gitlab":
		// It seems like gitlab is now correctly returning 404 instead of 403, keeping this for now for old on premise gitlab
		if err != nil {
			return strings.Contains(err.Error(), "Forbidden") || scmhelpers.IsScmNotFound(err)
		} else {
			return false
		}
	default:
		return scmhelpers.IsScmNotFound(err)
	}
}

// The code below is taken from https://github.com/antham/chyle/blob/master/chyle/git/git.go#L3
// Unfortunately it can't be imported since it uses an outdated version of go-git.

// node is a tree node in commit tree
type node struct {
	value  *object.Commit
	parent *node
}

// errNoDiffBetweenReferences is triggered when we can't
// produce any diff between 2 references
type errNoDiffBetweenReferences struct {
	from string
	to   string
}

func (e errNoDiffBetweenReferences) Error() string {
	return fmt.Sprintf(`can't produce a diff between %s and %s, check your range is correct by running "git log %[1]s..%[2]s" command`, e.from, e.to)
}

// errRepositoryPath is triggered when repository path can't be opened
type errRepositoryPath struct {
	path string
}

func (e errRepositoryPath) Error() string {
	return fmt.Sprintf(`check %q is an existing git repository path`, e.path)
}

// errReferenceNotFound is triggered when reference can't be
// found in git repository
type errReferenceNotFound struct {
	ref string
}

func (e errReferenceNotFound) Error() string {
	return fmt.Sprintf(`reference %q can't be found in git repository`, e.ref)
}

// errBrowsingTree is triggered when something wrong occurred during commit analysis process
var errBrowsingTree = fmt.Errorf("an issue occurred during tree analysis")

// FetchCommits retrieves commits in a reference range
func FetchCommits(repoPath, fromRef, toRef string) (*[]object.Commit, error) {
	rep, err := git.PlainOpen(repoPath)

	if err != nil {
		return nil, errRepositoryPath{repoPath}
	}

	fromCommit, err := resolveRef(fromRef, rep)

	if err != nil {
		return &[]object.Commit{}, err
	}

	toCommit, err := resolveRef(toRef, rep)

	if err != nil {
		return &[]object.Commit{}, err
	}

	var ok bool
	var commits *[]object.Commit

	exclusionList, err := buildOriginCommitList(fromCommit)

	if err != nil {
		return nil, err
	}

	if _, ok = exclusionList[toCommit.ID().String()]; ok {
		return nil, errNoDiffBetweenReferences{fromRef, toRef}
	}

	commits, err = findDiffCommits(toCommit, exclusionList)

	if err != nil {
		return nil, err
	}

	if len(*commits) == 0 {
		return nil, errNoDiffBetweenReferences{fromRef, toRef}
	}

	return commits, nil
}

// resolveRef gives hash commit for a given string reference
func resolveRef(refCommit string, repository *git.Repository) (*object.Commit, error) {
	hash := plumbing.Hash{}

	if strings.EqualFold(refCommit, "head") {
		head, err := repository.Head()

		if err == nil {
			return repository.CommitObject(head.Hash())
		}
	}

	iter, err := repository.References()

	if err != nil {
		return &object.Commit{}, errReferenceNotFound{refCommit}
	}

	err = iter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().Short() == refCommit {
			hash = ref.Hash()
		}

		return nil
	})

	if err == nil && !hash.IsZero() {
		return repository.CommitObject(hash)
	}

	hash = plumbing.NewHash(refCommit)

	if !hash.IsZero() {
		return repository.CommitObject(hash)
	}

	return &object.Commit{}, errReferenceNotFound{refCommit}
}

// buildOriginCommitList browses git tree from a given commit
// till root commit using kind of breadth first search algorithm
// and grab commit ID to a map with ID as key
func buildOriginCommitList(commit *object.Commit) (map[string]bool, error) {
	queue := append([]*object.Commit{}, commit)
	seen := map[string]bool{commit.ID().String(): true}

	for len(queue) > 0 {
		current := queue[0]
		queue = append([]*object.Commit{}, queue[1:]...)

		err := current.Parents().ForEach(
			func(c *object.Commit) error {
				if _, ok := seen[c.ID().String()]; !ok {
					seen[c.ID().String()] = true
					queue = append(queue, c)
				}

				return nil
			})

		if err != nil && err.Error() != plumbing.ErrObjectNotFound.Error() {
			return seen, errBrowsingTree
		}
	}

	return seen, nil
}

// findDiffCommits extracts commits that are no part of a given commit list
// using kind of depth first search algorithm to keep commits ordered
func findDiffCommits(commit *object.Commit, exclusionList map[string]bool) (*[]object.Commit, error) {
	commits := []object.Commit{}
	queue := append([]*node{}, &node{value: commit})
	seen := map[string]bool{commit.ID().String(): true}
	var current *node

	for len(queue) > 0 {
		current = queue[0]
		queue = append([]*node{}, queue[1:]...)

		if _, ok := exclusionList[current.value.ID().String()]; !ok {
			commits = append(commits, *(current.value))
		}

		err := current.value.Parents().ForEach(
			func(c *object.Commit) error {
				if _, ok := seen[c.ID().String()]; !ok {
					seen[c.ID().String()] = true
					n := &node{value: c, parent: current}
					queue = append([]*node{n}, queue...)
				}

				return nil
			})

		if err != nil && err.Error() != plumbing.ErrObjectNotFound.Error() {
			return &commits, errBrowsingTree
		}
	}

	return &commits, nil
}
