package gits

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/emirpasic/gods/sets/linkedhashset"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
)

type CommitInfo struct {
	Type        string
	Scope       string
	Description string
	group       *CommitGroup
}

type CommitGroup struct {
	Title string
	Order int
}

var (
	groupCounter          = 0
	undefinedGroupCounter = 0

	// ConventionalCommitTitles textual descriptions for
	// Conventional Commit types: https://conventionalcommits.org/
	ConventionalCommitTitles = map[string]*CommitGroup{
		"break":    createCommitGroup("BREAKING CHANGES"),
		"feat":     createCommitGroup("New Features"),
		"fix":      createCommitGroup("Bug Fixes"),
		"perf":     createCommitGroup("Performance Improvements"),
		"refactor": createCommitGroup("Code Refactoring"),
		"docs":     createCommitGroup("Documentation"),
		"test":     createCommitGroup("Tests"),
		"revert":   createCommitGroup("Reverts"),
		"style":    createCommitGroup("Styles"),
		"chore":    createCommitGroup("Chores"),
		"":         createCommitGroup("Other Changes"),
	}

	unknownKindOrder         = ConventionalCommitTitles[""].Order
	ConventionalCommitRegexp = regexp.MustCompile(`^([0-9A-Za-z-]+)(?:\(([0-9A-Za-z-]+)\))?(!)?: (.+)((?s:.*))`)
	BreakingChangeRegexp     = regexp.MustCompile(`(?m)^BREAKING CHANGE: (.*)`)
)

func createCommitGroup(title string) *CommitGroup {
	groupCounter++
	return &CommitGroup{
		Title: title,
		Order: groupCounter,
	}
}

// ParseCommit parses a conventional commit
// see: https://conventionalcommits.org/
func ParseCommit(message string) (*CommitInfo, *CommitInfo) {
	matches := ConventionalCommitRegexp.FindStringSubmatch(message)
	if matches == nil {
		return &CommitInfo{
			Description: message,
		}, nil
	}

	answer := &CommitInfo{
		Type:        matches[1],
		Scope:       matches[2],
		Description: matches[4],
	}
	breaking := BreakingChangeRegexp.FindStringSubmatch(matches[5])
	if breaking != nil {
		return answer, &CommitInfo{
			// Ugly to invent a special kind
			Type:        "break",
			Description: breaking[1],
		}
	} else if matches[3] == "!" {
		answer.Type = "break"
	}
	return answer, nil
}

func (c *CommitInfo) Group() *CommitGroup {
	if c.group == nil {
		title, found := ConventionalCommitTitles[strings.ToLower(c.Type)]
		if found {
			c.group = title
		} else {
			// Put unknown kinds first with the idea that if you invent
			// something for yourself it's probably important for you.
			undefinedGroupCounter--
			newGroup := &CommitGroup{
				Title: c.Type,
				Order: undefinedGroupCounter,
			}
			ConventionalCommitTitles[strings.ToLower(c.Type)] = newGroup
			c.group = newGroup
		}
	}
	return c.group
}

func (c *CommitInfo) Title() string {
	return c.Group().Title
}

func (c *CommitInfo) Order() int {
	return c.Group().Order
}

type GroupAndCommitInfos struct {
	group   *CommitGroup
	commits *linkedhashset.Set // duplicate commit messages should not show up in changelog
}

// GenerateMarkdown generates the markdown document for the commits
func GenerateMarkdown(releaseSpec *v1.ReleaseSpec, gitInfo *giturl.GitRepository, changelogSeparator string, prchangelog, includeprs bool) (string, error) {
	var hasCommitInfos bool

	groupAndCommits := map[int]*GroupAndCommitInfos{}

	issues := releaseSpec.Issues
	issueMap := map[string]*v1.IssueSummary{}
	for k := range issues {
		cp := issues[k]
		issueMap[cp.ID] = &cp
	}

	for i := range releaseSpec.Commits {
		cs := &releaseSpec.Commits[i]
		message := cs.Message
		if message != "" {
			ci, bc := ParseCommit(message)

			addCommitToGroup(gitInfo, cs, ci, issueMap, groupAndCommits)
			if bc != nil {
				addCommitToGroup(gitInfo, cs, bc, issueMap, groupAndCommits)
			}
			hasCommitInfos = true
		}
	}

	prs := releaseSpec.PullRequests

	var buffer bytes.Buffer
	if !hasCommitInfos && len(issues) == 0 && len(prs) == 0 {
		return "", nil
	}

	buffer.WriteString("## Changes in version " + releaseSpec.Version + "\n")

	hasTitle := false
	for i := undefinedGroupCounter; i <= unknownKindOrder; i++ {
		gac := groupAndCommits[i]
		if gac != nil && len(gac.commits.Values()) > 0 {
			group := gac.group
			if group != nil {
				legend := ""
				buffer.WriteString("\n")
				if i != unknownKindOrder || hasTitle {
					hasTitle = hasTitle || i != unknownKindOrder
					buffer.WriteString("### " + group.Title + "\n\n" + legend)
					if i == unknownKindOrder {
						buffer.WriteString("These commits did not use [Conventional Commits](https://conventionalcommits.org/) formatted messages:\n\n")
					}
				}
			}
			for _, msg := range gac.commits.Values() {
				buffer.WriteString(msg.(string))
			}
		}
	}

	if len(issues) > 0 {
		buffer.WriteString("\n### Issues\n\n")

		for k := range issues {
			buffer.WriteString(describeIssue(gitInfo, &issues[k], false, "", true))
		}
	}
	if len(prs) > 0 {
		if includeprs {
			buffer.WriteString("\n### Pull Requests\n\n")
		}
		for k := range prs {
			buffer.WriteString(describeIssue(gitInfo, &prs[k], prchangelog, changelogSeparator, includeprs))

		}
	}

	if len(releaseSpec.DependencyUpdates) > 0 {
		buffer.WriteString("\n### Dependency Updates\n\n")
		buffer.WriteString("| Component | New Version | Old Version |\n")
		buffer.WriteString("| --------- | ----------- | ----------- |\n")
		for i := range releaseSpec.DependencyUpdates {
			du := releaseSpec.DependencyUpdates[i]
			component := du.Component
			if du.URL != "" {
				component = fmt.Sprintf("[%s](%s)", component, du.URL)
			}
			buffer.WriteString(fmt.Sprintf("| %s | %s | %s |\n", component, du.ToVersion, du.FromVersion))
		}
	}
	return buffer.String(), nil
}

func addCommitToGroup(gitInfo *giturl.GitRepository, commits *v1.CommitSummary, ci *CommitInfo, issueMap map[string]*v1.IssueSummary, groupAndCommits map[int]*GroupAndCommitInfos) {
	description := "* " + describeCommit(gitInfo, commits, ci, issueMap) + "\n"
	group := ci.Group()
	gac := groupAndCommits[group.Order]
	if gac == nil {
		gac = &GroupAndCommitInfos{
			group:   group,
			commits: linkedhashset.New(),
		}
		groupAndCommits[group.Order] = gac
	}
	gac.commits.Add(description)
}

func describeIssue(info *giturl.GitRepository, issue *v1.IssueSummary, includeChangelog bool, separator string, includeDescription bool) string {
	changelog := ""
	if includeChangelog {
		parts := strings.SplitN(issue.Body, separator, 2)
		if len(parts) == 2 {
			changelog = "\n" + parts[1]
		}
	}
	if includeDescription {
		return "* " + describeIssueShort(issue) + issue.Title + describeUser(info, issue.User) + "\n" + changelog
	}
	return changelog
}

func describeIssueShort(issue *v1.IssueSummary) string {
	prefix := ""
	id := issue.ID
	if len(id) > 0 {
		// lets only add the hash prefix for numeric ids
		_, err := strconv.Atoi(id)
		if err == nil {
			prefix = "#"
		}
	}
	return "[" + prefix + issue.ID + "](" + issue.URL + ") "
}

func describeUser(info *giturl.GitRepository, user *v1.UserDetails) string {
	answer := ""
	if user != nil {
		userText := ""
		login := user.Login
		url := user.URL
		label := login
		if label == "" {
			label = user.Name
		}
		if url == "" && login != "" {
			url = stringhelpers.UrlJoin(info.HostURL(), login)
		}
		if url == "" {
			userText = label
		} else if label != "" {
			userText = "[" + label + "](" + url + ")"
		}
		if userText != "" {
			answer = " (" + userText + ")"
		}
	}
	return answer
}

func describeCommit(info *giturl.GitRepository, cs *v1.CommitSummary, ci *CommitInfo, issueMap map[string]*v1.IssueSummary) string {
	prefix := ""
	if ci.Scope != "" {
		prefix = ci.Scope + ": "
	}
	message := strings.TrimSpace(ci.Description)
	lines := strings.Split(message, "\n")

	// TODO add link to issue etc...
	user := cs.Author
	if user == nil {
		user = cs.Committer
	}
	issueText := ""
	for k := range cs.IssueIDs {
		issue := issueMap[cs.IssueIDs[k]]
		if issue != nil {
			issueText += " " + describeIssueShort(issue)
		}
	}
	return prefix + lines[0] + describeUser(info, user) + issueText
}
