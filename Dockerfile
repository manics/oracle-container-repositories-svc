# https://www.docker.com/blog/faster-multi-platform-builds-dockerfile-cross-compilation-guide/

###########################################################################
FROM --platform=$BUILDPLATFORM golang:1.18-alpine AS build

RUN apk add --no-cache make git

WORKDIR /src
COPY go.* *.go Makefile ./
# Needed for version info
COPY .git ./.git

ARG TARGETOS TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH make build

###########################################################################
FROM alpine:3.17

COPY --from=build /src/oci-container-repositories /bin

RUN adduser -S -D -H -h /app appuser
USER appuser

ENTRYPOINT ["/bin/oci-container-repositories"]

EXPOSE 8080
