# base go image
FROM golang:1.24.1-alpine3.21 AS builder

RUN mkdir /app

COPY . /app

WORKDIR /app 

RUN CGO_ENABLED=0 go build -o mailerApp ./cmd/api && \
    chmod +x /app/mailerApp


# build a tiny docker image
FROM alpine:3.21.3

WORKDIR /app 

COPY --from=builder /app/mailerApp mailerApp
COPY --from=builder /app/templates templates

CMD [ "/app/mailerApp" ]