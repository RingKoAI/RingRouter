# syntax=docker/dockerfile:1

# ── Stage 1: 构建前端 (web/dist) ─────────────────────────
FROM node:22-alpine AS web
WORKDIR /app/web
COPY web/package.json web/pnpm-lock.yaml* ./
# pnpm 优先，缺失时回退 npm
RUN corepack enable 2>/dev/null || true; \
    if [ -f pnpm-lock.yaml ]; then \
      pnpm install --frozen-lockfile; \
    else \
      npm install; \
    fi
COPY web/ .
RUN if [ -f pnpm-lock.yaml ]; then pnpm build; else npm run build; fi

# ── Stage 2: 编译 Go 二进制（嵌入 web/dist）──────────────
FROM golang:1.27-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=1 GOPROXY=https://goproxy.cn,direct \
    ALPINE_MIRROR=https://mirrors.aliyun.com
# gcc 用于 SQLite (mattn/go-sqlite3) 的 CGO 编译；使用国内源 + 重试
RUN sed -i "s#https://dl-cdn.alpinelinux.org/alpine#${ALPINE_MIRROR}/alpine#g" /etc/apk/repositories && \
    apk add --no-cache gcc musl-dev || apk add --no-cache gcc musl-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/web/dist ./web/dist
RUN go build -trimpath -ldflags="-s -w" -o /out/ringrouter .

# ── Stage 3: 精简运行镜像 ────────────────────────────────
FROM alpine:3.20
ENV ALPINE_MIRROR=https://mirrors.aliyun.com
RUN sed -i "s#https://dl-cdn.alpinelinux.org/alpine#${ALPINE_MIRROR}/alpine#g" /etc/apk/repositories && \
    apk add --no-cache ca-certificates tzdata && \
    addgroup -S ringrouter && adduser -S -G ringrouter ringrouter
WORKDIR /app
COPY --from=build /out/ringrouter /app/ringrouter
# 数据目录（sqlite 可选），以非 root 运行
RUN mkdir -p /app/data && chown -R ringrouter:ringrouter /app
USER ringrouter
EXPOSE 3000
ENTRYPOINT ["/app/ringrouter"]
