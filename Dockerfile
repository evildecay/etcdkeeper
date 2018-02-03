FROM golang:1.9-alpine as builder

RUN apk add -U git \
    && go get github.com/golang/dep/...    

WORKDIR /go/src/github.com/evildecay/etcdkeeper

ADD src ./
ADD Gopkg.* ./

RUN dep ensure -update \
    && go build -o etcdkeeper.bin httpserver/httpserver.go


FROM alpine:3.7

ENV HOST="127.0.0.1"
ENV PORT="8080"

WORKDIR /etcdkeeper
COPY --from=builder /go/src/github.com/evildecay/etcdkeeper/etcdkeeper.bin .
ADD assets assets

EXPOSE ${PORT}

ENTRYPOINT ./etcdkeeper.bin -h $HOST -p $PORT
