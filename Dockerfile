# Stage 1: 建置 Vue 前端
FROM node:20-alpine AS frontend-builder

WORKDIR /frontend
COPY frontend/package*.json ./
RUN npm ci --silent
COPY frontend/ .
RUN npm run build

# Stage 2: 建置 Go 後端
FROM golang:1.21-alpine AS backend-builder

WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .

# 上版關卡：測試沒過就不給 build。
# internal/websocket 是整場遊戲的整合測試（開房、免取名入場、拍照上傳、
# 重拍覆蓋、評選計分、重連、倒數收桌），全部跑在記憶體模式，不需要外部服務。
RUN CGO_ENABLED=0 go test -timeout 5m ./...

RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/main.go

# Stage 3: 最終映像
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=backend-builder /app/main .
COPY --from=frontend-builder /frontend/dist ./static

ENV ENV=production
EXPOSE 8081
CMD ["./main"]
