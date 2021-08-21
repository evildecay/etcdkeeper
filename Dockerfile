FROM golang:1.17-alpine3.14 as build

WORKDIR /app
ADD . /app
WORKDIR /app/src/etcdkeeper

ENV CGO_ENABLED=0

RUN go mod download
RUN go build -o ../../etcdkeeper -ldflags='-w -s' -a -tags netgo -installsuffix netgo main.go

FROM alpine:3.14.1

ENV HOST="0.0.0.0"
ENV PORT="8080"

RUN apk add --no-cache ca-certificates

WORKDIR /opt/etcdkeeper
COPY --from=build /app/etcdkeeper .
ADD assets assets

EXPOSE ${PORT}

ENTRYPOINT ["./etcdkeeper"]
CMD "-h $HOST -p $PORT"
