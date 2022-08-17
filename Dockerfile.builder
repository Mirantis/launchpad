FROM golang:1.19

WORKDIR /go/src/github.com/Mirantis/mcc

ENV GO111MODULE=on

ADD go.mod go.sum .
RUN go mod download -x

ADD . .
