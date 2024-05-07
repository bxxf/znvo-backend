FROM golang:1.22 as builder

WORKDIR /app

RUN apt-get update && apt-get install -y sqlite3 libsqlite3-dev

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -tags netgo -installsuffix cgo -ldflags '-w -extldflags "-static"' -o znvo-backend ./cmd/api/main.go

EXPOSE 40000

CMD ["./znvo-backend"]
