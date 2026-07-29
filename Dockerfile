# ===== Stage 1: build frontend (Svelte 5 + Vite) =====
# Pakai base glibc (bukan alpine/musl) + regenerate lockfile di dalam container: package-lock.json
# yang di-generate di Windows sering gagal resolve native binding (rolldown/esbuild/rollup) pas
# di-install ulang di Linux — https://github.com/npm/cli/issues/4828.
FROM node:22-bookworm-slim AS webbuilder
WORKDIR /web
COPY web2026/package.json ./
RUN npm install
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
