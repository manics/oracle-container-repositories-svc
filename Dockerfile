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

COPY --from=build /src/oracle-container-repositories-svc /bin

RUN adduser -S -D -H -h /app appuser
USER appuser

ENTRYPOINT ["/bin/oracle-container-repositories-svc"]

EXPOSE 8080
