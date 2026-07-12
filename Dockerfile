FROM golang:alpine AS totmodern
WORKDIR /go/src/github.com/devhdn-212/totmaster_api
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o app .


# Moving the binary to the 'final Image' to make it smaller
FROM alpine:latest as totmodernrelease
WORKDIR /app
RUN apk add tzdata
COPY --from=totmodern /go/src/github.com/devhdn-212/totmaster_api/app .
COPY --from=totmodern /go/src/github.com/devhdn-212/totmaster_api/.env /app/.env

ENV TZ=Asia/Jakarta
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

EXPOSE 6060
CMD ["./app"]