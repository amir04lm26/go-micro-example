# base go image
FROM golang:1.24.1-alpine3.21 AS builder

RUN mkdir /app

COPY . /app

WORKDIR /app 

RUN CGO_ENABLED=0 go build -o  loggerApp ./cmd/api && \
    chmod +x /app/loggerApp


# build a tiny docker image
FROM alpine:3.21.3

COPY --from=builder /app/loggerApp /app/loggerApp

CMD [ "/app/loggerApp" ]