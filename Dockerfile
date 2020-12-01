FROM gcr.io/jenkinsxio/jx-boot:3.0.759

RUN apk --no-cache add sed
    
COPY ./build/linux/jx-changelog /usr/bin/jx-changelog