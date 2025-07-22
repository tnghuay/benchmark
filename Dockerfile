# --- Stage 1: Build binary ---
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Cài các gói cần thiết
RUN apk add --no-cache git

# Copy toàn bộ mã nguồn vào container
COPY . .

# Tải module và build binary
RUN go mod tidy
RUN go build -o main .

# --- Stage 2: Tạo image nhỏ để chạy binary ---
FROM alpine:latest

WORKDIR /root/

# Copy binary từ stage builder
COPY --from=builder /app/main .

# Copy kubeconfig nếu cần (nếu chạy trong cluster thì không cần)
# COPY ./kubeconfig.yaml /root/.kube/config

# Cài chứng chỉ nếu cần truy cập Kubernetes API
RUN apk add --no-cache ca-certificates

# Entrypoint để chạy binary
ENTRYPOINT ["./main"]

