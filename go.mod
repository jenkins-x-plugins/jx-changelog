module github.com/jenkins-x-plugins/jx-changelog

require (
	github.com/andygrunwald/go-jira v1.13.0
	github.com/antham/chyle v1.11.0
	github.com/ghodss/yaml v1.0.0
	github.com/jenkins-x/go-scm v1.6.7
	github.com/jenkins-x/jx-api/v4 v4.0.25
	github.com/jenkins-x/jx-helpers/v3 v3.0.91
	github.com/jenkins-x/jx-logging/v3 v3.0.3
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.0.0
	github.com/stretchr/testify v1.6.1
	gopkg.in/src-d/go-git.v4 v4.13.1
	k8s.io/api v0.20.5 // indirect
	k8s.io/apimachinery v0.20.5
)

replace (
	k8s.io/api => k8s.io/api v0.20.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.3
	k8s.io/client-go => k8s.io/client-go v0.20.3
)

go 1.15
