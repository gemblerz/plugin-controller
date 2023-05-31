FROM 1.20.4-alpine3.17 as base
WORKDIR /app

FROM base as builder
COPY . /app/
RUN go mod download \
  && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./out/plugin-controller cmd/controller/main.go \
  && chmod +x ./out/plugin-controller

FROM base
COPY --from=builder /app/out/plugin-controller /app/plugin-controller

ENTRYPOINT ["/app/plugin-controller"]
