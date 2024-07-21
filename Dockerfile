#
# This is for local development only.
# See Dockerfile.goreleaser for the image published on release or staging.
#

FROM golang:1.21 as base

SHELL ["/bin/bash", "-o", "pipefail", "-euxc"]

WORKDIR /opt/app/api

ENV AIR_SHA256 3842f9a86304e06f68f61555ce303cb426b450a7be8ee14020cbd149a68008d0

RUN export GO_BINPATH="$(go env GOPATH)/bin" \
    && echo $GO_BINPATH \
    && curl -sSfL -o "$GO_BINPATH/air" https://github.com/cosmtrek/air/releases/download/v1.40.2/air_1.40.2_linux_amd64 \
    && echo "$AIR_SHA256 $GO_BINPATH/air" | sha256sum --check --strict \
    && chmod +x "$GO_BINPATH/air"

CMD ["air"]
