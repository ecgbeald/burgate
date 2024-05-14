FROM golang:1.22-alpine

WORKDIR /app

COPY go.mod go.sum order/order-air.toml ./

RUN go install github.com/cosmtrek/air@latest

RUN go mod download

COPY . .

CMD ["air", "-c", "order/order-air.toml"]