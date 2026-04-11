# 多阶段构建：运行阶段使用 Alpine，避免 Debian apt 在部分网络环境下失败。
FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/server ./server
COPY --from=build /src/web ./web
RUN chmod 755 /app/server
EXPOSE 8080
USER nobody
ENTRYPOINT ["/app/server"]
