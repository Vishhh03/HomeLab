# Stage 1: Build the binary
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o fitness-api .

# Stage 2: Run the binary
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/fitness-api .
COPY --from=builder /app/index.html .
COPY --from=builder /app/static ./static
EXPOSE 8081
# Run the binary
CMD ["./fitness-api"]
