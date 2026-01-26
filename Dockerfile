# Build frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Build backend
FROM golang:1.22-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY backend/ ./
RUN CGO_ENABLED=1 go build -o weeky ./cmd/server

# Final image
FROM alpine:3.19
RUN apk add --no-cache ca-certificates libc6-compat
WORKDIR /app

COPY --from=backend-builder /app/weeky .
COPY --from=frontend-builder /app/backend/dist ./dist

ENV PORT=8080
ENV DB_PATH=/app/data/weeky.db

EXPOSE 8080

VOLUME ["/app/data", "/app/templates"]

CMD ["./weeky"]
