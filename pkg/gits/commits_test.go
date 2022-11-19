//go:build unit

package gits_test

import (
	"testing"

	"github.com/jenkins-x-plugins/jx-changelog/pkg/gits"
	"github.com/stretchr/testify/assert"
)

func TestParseCommits(t *testing.T) {
	t.Parallel()
	assertParseCommit(t, "something regular", &gits.CommitInfo{
		Description: "something regular",
	}, nil)
	assertParseCommit(t, "feat: cheese", &gits.CommitInfo{
		Type:        "feat",
		Description: "cheese",
	}, nil)
	assertParseCommit(t, "feat(beer): wine is good too", &gits.CommitInfo{
		Type:        "feat",
		Scope:       "beer",
		Description: "wine is good too",
	}, nil)
	assertParseCommit(t, "FOO 123: beer rules!", &gits.CommitInfo{
		Description: "FOO 123: beer rules!",
	}, nil)
	assertParseCommit(t, "FOO!: beer rules", &gits.CommitInfo{
		Type:        "break",
		Description: "beer rules",
	}, nil)
	assertParseCommit(t, `FOO-123!: beer rules
	lorem ipsum
BREAKING CHANGE: beer is out!
`,
		&gits.CommitInfo{
			Type:        "FOO-123",
			Description: "beer rules",
		},
		&gits.CommitInfo{
			Type:        "break",
			Description: "beer is out!",
		})
	assertParseCommit(t, "The nice url http://jenkins-x.io",
		&gits.CommitInfo{
			Description: "The nice url http://jenkins-x.io",
		}, nil)
}

func assertParseCommit(t *testing.T, input string, expected *gits.CommitInfo, expectedBreaking *gits.CommitInfo) {
	info, breaking := gits.ParseCommit(input)
	assert.NotNil(t, info)
	assert.Equal(t, expected.Type, info.Type, "Kind for Commit %s", info)
	assert.Equal(t, expected.Scope, info.Scope, "Feature for Commit %s", info)
	assert.Equal(t, expected.Description, info.Description, "Message for Commit %s", info)
	assert.Equal(t, expected, info, "CommitInfo for Commit %s", info)
	assert.Equal(t, expectedBreaking, breaking, "Breaking CommitInfo for Commit %s", breaking)
}
