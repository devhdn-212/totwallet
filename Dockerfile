# ===== Stage 1: build frontend (Svelte 5 + Vite) =====
FROM node:22-alpine AS webbuilder
WORKDIR /web
COPY web2026/package.json web2026/package-lock.json ./
RUN npm ci
COPY web2026/ .
RUN npm run build

# ===== Stage 2: build backend (Go) =====
FROM golang:alpine AS totwallet
WORKDIR /go/src/github.com/devhdn-212/totwallet
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o app .

# ===== Stage 3: final runtime image =====
FROM alpine:latest as totwalletrelease
WORKDIR /app
RUN apk add tzdata
COPY --from=totwallet /go/src/github.com/devhdn-212/totwallet/app .
COPY --from=totwallet /go/src/github.com/devhdn-212/totwallet/.env /app/.env
COPY --from=webbuilder /web/dist /app/web2026/dist

ENV TZ=Asia/Jakarta
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

EXPOSE 6167
CMD ["./app"]
