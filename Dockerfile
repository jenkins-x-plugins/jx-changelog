FROM centos:7

RUN yum install -y git

ENTRYPOINT ["jx-changelog"]

COPY ./build/linux/jx-changelog /usr/bin/jx-changelog