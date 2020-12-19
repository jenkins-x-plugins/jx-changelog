module github.com/jenkins-x-plugins/jx-changelog

require (
	github.com/andygrunwald/go-jira v1.13.0
	github.com/antham/chyle v1.11.0
	github.com/cpuguy83/go-md2man v1.0.10
	github.com/ghodss/yaml v1.0.0
	github.com/jenkins-x/go-scm v1.5.199
	github.com/jenkins-x/jx-api/v4 v4.0.14
	github.com/jenkins-x/jx-helpers/v3 v3.0.41
	github.com/jenkins-x/jx-logging/v3 v3.0.2
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	gopkg.in/src-d/go-git.v4 v4.13.1
	k8s.io/apimachinery v0.19.2

)

replace github.com/jenkins-x/go-scm => github.com/jstrachan/go-scm v1.5.1-0.20201201055349-98694c17b897

go 1.15
