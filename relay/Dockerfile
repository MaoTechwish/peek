FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/peek-relay .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/peek-relay /app/peek-relay
EXPOSE 8080
CMD ["/app/peek-relay"]
