# ===== 第一阶段：构建前端 =====
FROM node:18-alpine AS frontendbuilder
WORKDIR /app
RUN npm install -g pnpm
COPY ui/package.json ui/pnpm-lock.yaml ./ui/
RUN cd ui && pnpm install --frozen-lockfile
COPY ui/ ./ui/
RUN cd ui && CI=false pnpm build
RUN mkdir -p /app/public && cp -r ui/build/* public/

# ===== 第二阶段：构建后端 =====
FROM golang:1.23-alpine AS binarybuilder
RUN apk --no-cache add git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontendbuilder /app/public /app/public
RUN go build -o nav .

# ===== 第三阶段：最终镜像 =====
FROM alpine:3.20
ENV TZ="Asia/Shanghai"
RUN apk --no-cache add ca-certificates tzdata && \
    cp "/usr/share/zoneinfo/$TZ" /etc/localtime && \
    echo "$TZ" > /etc/timezone

WORKDIR /app
COPY --from=binarybuilder /app/nav /app/
#COPY --from=binarybuilder /app/public /app/public

VOLUME ["/app/data"]
EXPOSE 6412
ENTRYPOINT ["/app/nav"]