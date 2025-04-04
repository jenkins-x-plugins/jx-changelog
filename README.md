# jx changelog

[![Documentation](https://godoc.org/github.com/jenkins-x-plugins/jx-changelog?status.svg)](https://pkg.go.dev/mod/github.com/jenkins-x-plugins/jx-changelog)
[![Go Report Card](https://goreportcard.com/badge/github.com/jenkins-x-plugins/jx-changelog)](https://goreportcard.com/report/github.com/jenkins-x-plugins/jx-changelog)
[![Releases](https://img.shields.io/github/release-pre/jenkins-x/jx-changelog.svg)](https://github.com/jenkins-x-plugins/jx-changelog/releases)
[![Apache](https://img.shields.io/badge/license-Apache-blue.svg)](https://github.com/jenkins-x-plugins/jx-changelog/blob/master/LICENSE)
[![Slack Status](https://img.shields.io/badge/slack-join_chat-white.svg?logo=slack&style=social)](https://slack.k8s.io/)

`jx-changelog` is a small command line tool for generating release Changelog files

## Debug
To debug jx changelog inside a Running container:
First modify you pipeline by editing `release.yaml`  in your project and add
```yaml
script: |
  #!/usr/bin/env sh
  # default script content before `jx changelog create` in https://github.com/jenkins-x/jx3-pipeline-catalog/blob/master/tasks/gradle/release.yaml or similar
  while sleep 10; do echo "waiting for debug"; done
```
to the step `promote-changelog`

build your version of jx changelog locally, and copy it inside the container
```shell script
make build
kubectl cp ./build/jx-changelog release-xxxxxxxx:/ -c step-promote-changelog
```
once the pipeline reaches the promote-changelog step, exec into the container:
```shell script
kubectl exec -it release-xxxxxxxx -c step-promote-changelog -- sh
```
and run:
```shell script
# apk update; apk add go doesn't work anymore, version of alpine too old
wget https://go.dev/dl/go1.22.1.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.22.1.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
go install github.com/go-delve/delve/cmd/dlv@latest
```
then debug your binary using dlv
```shell script
source /workspace/source/.jx/variables.sh # copied from pipeline
/tekton/home/go/bin/dlv --listen=:2345 --headless=true --api-version=2 exec /jx-changelog create -- --version v${VERSION}
```
redirect traffic from your port 2345 to the container in another terminal
```shell script
kubectl port-forward release-xxxxxxxx 2345
```
attach your debugger and happy debugging.

Do not forget to `make build` and `kubectl cp` after each change

## Commands

See the [jx-changelog command reference](https://jenkins-x.io/v3/develop/reference/jx/changelog/)

