module github.com/jenkins-x-plugins/jx-changelog

require (
	github.com/andygrunwald/go-jira v1.13.0
	github.com/antham/chyle v1.11.0
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/ghodss/yaml v1.0.0
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/jenkins-x/go-scm v1.6.18
	github.com/jenkins-x/jx-api/v4 v4.0.28
	github.com/jenkins-x/jx-helpers/v3 v3.0.104
	github.com/jenkins-x/jx-logging/v3 v3.0.3
	github.com/onsi/gomega v1.8.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/shurcooL/githubv4 v0.0.0-20191102174205-af46314aec7b // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	gopkg.in/src-d/go-git.v4 v4.13.1
	k8s.io/api v0.20.6 // indirect
	k8s.io/apimachinery v0.20.6
)

replace (
	k8s.io/api => k8s.io/api v0.20.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.3
	k8s.io/client-go => k8s.io/client-go v0.20.3
)

go 1.15
