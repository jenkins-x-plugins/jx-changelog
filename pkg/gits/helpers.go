package gits

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
)

// GetRevisionBeforeDateText returns the revision before the given date in format "MonthName dayNumber year"
func GetRevisionBeforeDateText(g gitclient.Interface, dir, dateText string) (string, error) {
	branch, err := gitclient.Branch(g, dir)
	if err != nil {
		return "", err
	}
	return g.Command(dir, "rev-list", "-1", "--before=\""+dateText+"\"", "--max-count=1", branch)
}

// GetCommitPointedToByLatestTag return the SHA of the commit pointed to by the latest git tag as well as the tag name
// for the git repo in dir
func GetCommitPointedToByLatestTag(g gitclient.Interface, dir, prefix string) (string, string, error) {
	tagList, err := NTags(g, dir, 1, prefix)
	if err != nil {
		return "", "", fmt.Errorf("getting commit pointed to by latest tag in %s: %w", dir, err)
	}
	if len(tagList) == 0 {
		return "", "", nil
	}
	return GetCommitForTagSha(g, dir, tagList[0][0], tagList[0][1])
}

func GetCommitForTagSha(g gitclient.Interface, dir, tagSHA, tagName string) (string, string, error) {
	commitSHA, err := g.Command(dir, "rev-list", "-n", "1", tagSHA)
	if err != nil {
		return "", "", fmt.Errorf("running for git rev-list -n 1 %s: %w", tagSHA, err)
	}
	return commitSHA, tagName, err
}

// NTags return the SHA and tag name of n first tags in reverse chronological order from the repository at the given directory.
// If N tags doesn't exist the available tags are returned without an error.
func NTags(g gitclient.Interface, dir string, n int, prefix string) ([][]string, error) {
	args := []string{
		"for-each-ref",
		"--sort=-creatordate",
		"--format=%(objectname)%00%(refname:short)",
		fmt.Sprintf("--count=%d", n),
		"refs/tags/" + prefix + "*",
	}
	out, err := g.Command(dir, args...)
	if err != nil {
		return nil, fmt.Errorf("running git %s: %w", strings.Join(args, " "), err)
	}

	tagList := strings.Split(out, "\n")
	res := make([][]string, len(tagList))
	for i, tag := range tagList {
		fields := strings.Split(tag, "\x00")

		if len(fields) != 2 {
			return nil, fmt.Errorf("Unexpected format for returned tag and sha: '%s'", tagList[n-1])
		}
		res[i] = fields
	}
	return res, nil
}

// GetFirstCommitSha returns the sha of the first commit
func GetFirstCommitSha(g gitclient.Interface, dir string) (string, error) {
	return g.Command(dir, "rev-list", "--max-parents=0", "HEAD")
}

// FilterTags returns all tags from the repository at the given directory that match the filter
func FilterTags(g gitclient.Interface, dir, filter string) ([]string, error) {
	args := []string{"tag"}
	if filter != "" {
		args = append(args, "--list", filter)
	}
	text, err := g.Command(dir, args...)
	if err != nil {
		return nil, err
	}
	text = strings.TrimSuffix(text, "\n")
	split := strings.Split(text, "\n")
	// Split will return the original string if it can't split it, and it may be empty
	if len(split) == 1 && split[0] == "" {
		return make([]string, 0), nil
	}
	return split, nil
}
