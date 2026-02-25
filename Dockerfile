# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# Stage 2: Build Go backend (no CGO required)
FROM golang:1.22-alpine AS backend
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o ansible-frontend .

# Stage 3: Minimal runtime image
FROM alpine:3.19
RUN apk add --no-cache ca-certificates openssh-client
WORKDIR /app
COPY --from=backend /app/ansible-frontend ./ansible-frontend
COPY --from=backend /app/frontend/dist ./frontend/dist
EXPOSE 8080
CMD ["./ansible-frontend"]
