#
# This is for local development only.
# See Dockerfile.goreleaser for the image published on release.
#

FROM golang:1.18 as base

FROM base as dev

RUN curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

WORKDIR /opt/app/api
CMD ["air"]
