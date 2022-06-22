#
# This is for local development only.
# See Dockerfile.goreleaser for the image published on release.
#

ARG GO_VERSION=1.18

FROM golang:${GO_VERSION} as build

WORKDIR /go/src

COPY . .

RUN go build

FROM alpine:3

COPY --from=build /go/src/developers-italia-api /usr/local/bin/developers-italia-api

ENTRYPOINT ["/usr/local/bin/developers-italia-api"]
