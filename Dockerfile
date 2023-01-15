# https://www.docker.com/blog/faster-multi-platform-builds-dockerfile-cross-compilation-guide/
###########################################################################

FROM --platform=$BUILDPLATFORM golang:1.18-alpine AS build
WORKDIR /src
COPY go.* *.go ./
ARG TARGETOS TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /out/oci-container-repositories .

###########################################################################

FROM alpine:3.17
COPY --from=build /out/oci-container-repositories /bin
ENTRYPOINT ["/bin/oci-container-repositories"]

EXPOSE 8080
