FROM golang:1-alpine as builder

ENV CGO_ENABLED=0

WORKDIR /workdir

COPY . .

RUN go build -v -o /workdir/shady-bot ./cmd/server/

FROM alpine:3

RUN apk --no-cache add ca-certificates

COPY --from=builder /workdir/shady-bot /bin/shady-bot

CMD [ "/bin/shady-bot" ]