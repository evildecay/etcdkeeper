FROM golang:1.8-alpine as builder

ADD . /go/src/github.com/evildecay/etcdkeeper

RUN apk add -U git \
    && cd /go/src/github.com/evildecay/etcdkeeper \
    && go get github.com/golang/dep/... \
    && dep ensure -update \
    && go build -o etcdkeeper.bin src/httpserver/httpserver.go

FROM alpine:3.6

WORKDIR /etcdkeeper
COPY --from=builder /go/src/github.com/evildecay/etcdkeeper/etcdkeeper.bin .
ADD assets assets

CMD ./etcdkeeper.bin
