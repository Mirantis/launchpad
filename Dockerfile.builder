FROM golang:1.19

WORKDIR /go/src/github.com/Mirantis/mcc

ENV GO111MODULE=on

RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.48.0

ADD go.mod go.sum .
RUN go mod download -x

ADD . .
