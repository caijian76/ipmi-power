FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY * ./
RUN go mod download

RUN CGO_ENABLED=0 go build -ldflags '-w -s -extldflags "-static"' -o ipmi-power

FROM alpine:3.18 AS runner

RUN apk add --no-cache tzdata \
    && ln -snf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo Asia/Shanghai > /etc/timezone

COPY --from=builder /src/ipmi-power /ipmi-power
COPY --from=builder /src/config.json /config.json

ENTRYPOINT ["/ipmi-power"]