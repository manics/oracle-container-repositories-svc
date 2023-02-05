# https://www.docker.com/blog/faster-multi-platform-builds-dockerfile-cross-compilation-guide/

###########################################################################
FROM --platform=$BUILDPLATFORM golang:1.18-alpine AS build

RUN apk add --no-cache make git

WORKDIR /src
COPY . ./

ARG TARGETOS TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH make build

###########################################################################
FROM alpine:3.17

COPY --from=build /src/binderhub-container-registry-helper /bin

RUN adduser -S -D -H -h /app appuser
USER appuser

ENTRYPOINT ["/bin/binderhub-container-registry-helper"]

EXPOSE 8080
