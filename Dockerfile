# https://www.docker.com/blog/faster-multi-platform-builds-dockerfile-cross-compilation-guide/

###########################################################################
FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.20-alpine AS build

RUN apk add --no-cache make git

WORKDIR /src
COPY . ./

ARG TARGETOS TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH make build

###########################################################################
FROM docker.io/library/alpine:3.18

COPY --from=build /src/binderhub-amazon /src/binderhub-oracle /bin/

RUN adduser -S -D -H -h /app appuser
USER appuser

# CMD [ "binderhub-amazon" ]
# CMD [ "binderhub-oracle" ]

EXPOSE 8080
