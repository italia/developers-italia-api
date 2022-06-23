#
# This is for local development only.
# See Dockerfile.goreleaser for the image published on release.
#

ARG GO_VERSION=1.18

FROM golang:${GO_VERSION} as build

WORKDIR /go/src/developers-italia-api

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o app .

FROM alpine:latest

WORKDIR /app

COPY --from=build /go/src/developers-italia-api/app .

EXPOSE 3000

CMD ["./app"]
