FROM golang:1.21

RUN apt-get update

WORKDIR /go/src/github.com/Mirantis/mcc

ENV GO111MODULE=on

ADD go.mod go.sum ./

RUN go mod download
