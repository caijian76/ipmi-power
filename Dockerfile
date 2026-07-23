FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY * ./
RUN go mod download

RUN CGO_ENABLED=0 go build -ldflags '-w -s -extldflags "-static"' -o ipmi-power main.go

FROM scratch

COPY --from=builder /src/ipmi-power /ipmi-power
COPY --from=builder /src/config.json /config.json

ENTRYPOINT ["/ipmi-power"]