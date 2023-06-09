FROM golang:1.20.4-alpine3.17 as base
WORKDIR /

FROM base as builder
WORKDIR /code
COPY . /code/
RUN go mod download \
  && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./out/plugin-controller cmd/controller/main.go \
  && chmod +x ./out/plugin-controller

FROM alpine:3.17
COPY --from=builder /code/out/plugin-controller /plugin-controller

ENTRYPOINT ["/plugin-controller"]
