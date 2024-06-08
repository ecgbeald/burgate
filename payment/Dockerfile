FROM golang:1.22-alpine

WORKDIR /app

COPY go.mod go.sum payment/payment-air.toml ./

RUN go install github.com/air-verse/air@latest

RUN go mod download

COPY . .

CMD ["air", "-c", "payment/payment-air.toml"]