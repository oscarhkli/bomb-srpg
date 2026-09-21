# Stage 1: build the frontend
FROM node:26-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: build the Go binary
FROM golang:1.26.5-alpine AS backend-builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/bin/srpg-web ./cmd/srpg-web

# Stage 3: minimal runtime
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=backend-builder /app/bin/srpg-web ./srpg-web
COPY --from=frontend-builder /app/web/dist ./web/dist
USER nonroot:nonroot
EXPOSE 8080
CMD ["./srpg-web"]
