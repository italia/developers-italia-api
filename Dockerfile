#
# This is for local development only.
# See Dockerfile.goreleaser for the image published on release or staging.
#

FROM golang:1.27@sha256:7543a96ce82c8e9003cae079ee3e0bc5b7799df8eed2a041e403af0d31fa4e67 AS base

SHELL ["/bin/bash", "-o", "pipefail", "-euxc"]

WORKDIR /opt/app/api

CMD ["go", "run", "."]
