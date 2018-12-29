############################
# STEP 1 build executable binary
############################
FROM golang:alpine AS builder

# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git
COPY . $GOPATH/src/github.com/krantius/definitely-not-raft
WORKDIR $GOPATH/src/github.com/krantius/definitely-not-raft

#ARG GO111MODULE=on
# Fetch dependencies.
# Using go get.
RUN go get -d -v
# Build the binary.
RUN GOOS=linux GOARCH=amd64 go build -o /go/bin/doit
############################
# STEP 2 build a small image
############################
FROM scratch
# Copy our static executable.
COPY --from=builder /go/bin/doit /go/bin/doit
# Run the hello binary.
ENTRYPOINT ["/go/bin/doit"]
