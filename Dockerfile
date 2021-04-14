FROM ghcr.io/jenkins-x/jx-boot:3.2.39

RUN apk --no-cache add sed
    
COPY ./build/linux/jx-changelog /usr/bin/jx-changelog
