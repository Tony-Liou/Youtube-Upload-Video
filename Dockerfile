# syntax=docker/dockerfile:1
FROM golang:1.17

WORKDIR /go/src/app
COPY . .

RUN go build -o app

RUN apt update && apt install streamlink -y

EXPOSE 8080

CMD ["./app"]