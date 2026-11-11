# Build Stage
FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache build-base git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compilação estática com remoção de símbolos de debug
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w -extldflags '-static'" -o /bin/casbin-engine cmd/api/main.go

# Production Stage
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /bin/casbin-engine /app/casbin-engine
COPY config/ /app/config/

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/casbin-engine"]
