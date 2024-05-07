FROM golang:1.22 as builder

WORKDIR /app

RUN apt-get update && apt-get install -y sqlite3 libsqlite3-dev

COPY go.mod go.sum ./

COPY . .

RUN go mod download

RUN go build -o znvo-backend ./cmd/api/main.go

EXPOSE 40000

CMD ["./znvo-backend"]
