### Linux

```shell
curl -L https://github.com/jenkins-x-plugins/jx-changelog/releases/download/v{{.Version}}/jx-changelog-linux-amd64.tar.gz | tar xzv 
sudo mv jx-changelog /usr/local/bin
```

### macOS

```shell
curl -L  https://github.com/jenkins-x-plugins/jx-changelog/releases/download/v{{.Version}}/jx-changelog-darwin-amd64.tar.gz | tar xzv
sudo mv jx-changelog /usr/local/bin
```

