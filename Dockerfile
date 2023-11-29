FROM golang:1.20-alpine as build

WORKDIR /app
COPY . .

RUN apk add git openssh

WORKDIR /app

RUN go mod download \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server .

FROM alpine

WORKDIR /app
COPY --from=build /app/server .

EXPOSE 8080

CMD ["./server"]
