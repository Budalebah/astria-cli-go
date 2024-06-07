# builder image
FROM golang:1.22 AS builder

# set current working directory
WORKDIR /build

# go deps
COPY go.mod go.sum ./
RUN go mod download

# copy source
COPY . .

# build the Go app for the target platform
ARG TARGETOS
ARG TARGETARCH
ARG VERSION
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -X github.com/astria/astria-cli-go/cmd.version=${VERSION}" \
    -o /build/astria-go .

# final stage
FROM alpine

# install ca certificates
RUN apk --no-cache add ca-certificates

# copy the built binary file from the builder stage
COPY --from=builder /build/astria-go /build/astria-go

# expose port 8080 to the outside world
# FIXME - the ports are for devrunner. docker image probably isn't ideal for running devrunner,
# but i still don't want to have a half working docker image.
# like would i have to expose a ton of ports here? what if they use a different port?
EXPOSE 8080

# command to run the executable
ENTRYPOINT ["/build/astria-go"]
