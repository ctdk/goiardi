FROM golang:1.8

# A Dockerfile that lets you run the local source tree of goiardi easily
# locally. See https://hub.docker.com/r/ctdk/goiardi/ for official goiardi
# docker images and https://github.com/ctdk/goiardi-docker for the sources of
# those docker images.

RUN mkdir -p /go/src/github.com/ctdk/goiardi
RUN mkdir -p /etc/goiardi
RUN mkdir -p /var/lib/goiardi/lfs

COPY ./etc/docker-goiardi.conf /etc/goiardi/goiardi.conf

WORKDIR /go/src/github.com/ctdk/goiardi

# this will ideally be built by the ONBUILD below ;)
CMD ["goiardi", "-c", "/etc/goiardi/goiardi.conf"]

COPY . /go/src/github.com/ctdk/goiardi
RUN go get -v -d
RUN go install github.com/ctdk/goiardi
