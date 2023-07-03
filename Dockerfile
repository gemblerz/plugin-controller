FROM golang:1.20.4-alpine3.17 as base
WORKDIR /

FROM base as builder
ARG TARGETARCH
WORKDIR /code
COPY . /code/
RUN go mod download \
  && make plugin-controller-${TARGETARCH} \
  && chmod +x ./out/plugin-controller

FROM alpine:3.17
COPY --from=builder /code/out/plugin-controller /plugin-controller

ENTRYPOINT ["/plugin-controller"]
