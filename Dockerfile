# 1단계: 프론트엔드 빌드
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# 2단계: 백엔드 빌드 (CGO 불필요 - pure Go SQLite)
FROM golang:1.24-alpine AS backend-builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend-builder /app/backend/dist ./dist
RUN CGO_ENABLED=0 go build -o weeky -ldflags="-s -w" ./cmd/server

# 3단계: 최소 런타임
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend-builder /app/weeky .
COPY --from=backend-builder /app/dist ./dist

EXPOSE 8080
VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://localhost:8080/health || exit 1

CMD ["./weeky"]
