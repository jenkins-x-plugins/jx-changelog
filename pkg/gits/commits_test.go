//go:build unit
// +build unit

package gits_test

import (
	"testing"

	"github.com/jenkins-x-plugins/jx-changelog/pkg/gits"
	"github.com/stretchr/testify/assert"
)

func TestParseCommits(t *testing.T) {
	t.Parallel()
	assertParseCommit(t, "something regular", &gits.CommitInfo{
		Message: "something regular",
	}, nil)
	assertParseCommit(t, "feat: cheese", &gits.CommitInfo{
		Kind:    "feat",
		Message: "cheese",
	}, nil)
	assertParseCommit(t, "feat(beer): wine is good too", &gits.CommitInfo{
		Kind:    "feat",
		Feature: "beer",
		Message: "wine is good too",
	}, nil)
	assertParseCommit(t, "FOO 123: beer rules!", &gits.CommitInfo{
		Message: "FOO 123: beer rules!",
	}, nil)
	assertParseCommit(t, "FOO!: beer rules", &gits.CommitInfo{
		Kind:    "break",
		Message: "beer rules",
	}, nil)
	assertParseCommit(t, `FOO-123!: beer rules
	lorem ipsum
BREAKING CHANGE: beer is out!
`,
		&gits.CommitInfo{
			Kind:    "FOO-123",
			Message: "beer rules",
		},
		&gits.CommitInfo{
			Kind:    "break",
			Message: "beer is out!",
		})
	assertParseCommit(t, "The nice url http://jenkins-x.io",
		&gits.CommitInfo{
			Message: "The nice url http://jenkins-x.io",
		}, nil)
}

func assertParseCommit(t *testing.T, input string, expected *gits.CommitInfo, expectedBreaking *gits.CommitInfo) {
	info, breaking := gits.ParseCommit(input)
	assert.NotNil(t, info)
	assert.Equal(t, expected.Kind, info.Kind, "Kind for Commit %s", info)
	assert.Equal(t, expected.Feature, info.Feature, "Feature for Commit %s", info)
	assert.Equal(t, expected.Message, info.Message, "Message for Commit %s", info)
	assert.Equal(t, expected, info, "CommitInfo for Commit %s", info)
	assert.Equal(t, expectedBreaking, breaking, "Breaking CommitInfo for Commit %s", breaking)
}
