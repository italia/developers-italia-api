#
# This is for local development only.
# See Dockerfile.goreleaser for the image published on release or staging.
#

FROM golang:1.25@sha256:699337d620559a59b4a2bb298ad59611e535d2ee755a34cf2d2a98f37578dc80 AS base

SHELL ["/bin/bash", "-o", "pipefail", "-euxc"]

WORKDIR /opt/app/api

CMD ["go", "run", "."]
