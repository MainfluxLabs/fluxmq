FROM golang:1.8-alpine AS builder
WORKDIR /go/src/github.com/mainflux/fluxmq
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo -o fluxmq

FROM scratch
COPY --from=builder /go/src/github.com/mainflux/fluxmq /fluxmq /
EXPOSE 1883
ENTRYPOINT ["/fluxmq"]
