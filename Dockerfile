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

# Create a user 'etcdkeeper' member of group 'etcdkeeper'
RUN addgroup -S etcdkeeper && \
    adduser -S -D -h /etcdkeeper -G etcdkeeper etcdkeeper

WORKDIR /opt/etcdkeeper
COPY --from=build --chown=etcdkeeper:etcdkeeper /app/etcdkeeper .
ADD --chown=etcdkeeper:etcdkeeper assets assets

EXPOSE ${PORT}
USER etcdkeeper

ENTRYPOINT ["./etcdkeeper"]
CMD "-h $HOST -p $PORT"
