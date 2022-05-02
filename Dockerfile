FROM ghcr.io/jenkins-x/jx-boot:latest

RUN apk --no-cache add sed
    
COPY ./build/linux/jx-changelog /usr/bin/jx-changelog
