FROM golang:1.8-alpine as builder

ADD . /go/src/github.com/evildecay/etcdkeeper3

RUN apk add -U git \
    && cd /go/src/github.com/evildecay/etcdkeeper3 \
    && go get github.com/golang/dep/... \
    && dep ensure -update \
    && go build -o etcdkeeper3.bin httpserver3.go

FROM alpine:3.6

WORKDIR /etcdkeeper3
COPY --from=builder /go/src/github.com/evildecay/etcdkeeper3/etcdkeeper3.bin .
ADD etcdkeeper3 /etcdkeeper3

CMD ./etcdkeeper3.bin
