FROM ghcr.io/jenkins-x/jx-boot:latest as builder

FROM alpine:3.16.0
COPY --from=builder /usr/bin/jx /usr/bin/jx
COPY --from=builder /root/.jx/plugins/bin/jx-gitops* /root/.jx3/plugins/bin/
RUN apk --no-cache add sed git
COPY ./build/linux/jx-changelog /usr/bin/jx-changelog
