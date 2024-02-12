FROM golang:1.21

RUN apt-get update

WORKDIR /go/src/github.com/Mirantis/mcc

ENV GO111MODULE=on
ENV CGO_ENABLED=0

ADD go.mod go.sum ./

RUN go mod download
