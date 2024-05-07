FROM golang:1.22 as builder

WORKDIR /app

RUN apt-get update && apt-get install -y sqlite3 libsqlite3-dev

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -tags netgo -installsuffix cgo -ldflags '-w -extldflags "-static"' -o myapp ./cmd/api/main.go

FROM debian:buster

RUN apt-get update && apt-get install -y sqlite3 libsqlite3-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

COPY --from=builder /app/myapp .

EXPOSE 40000

CMD ["./myapp"]
