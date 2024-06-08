FROM golang:1.22-alpine

WORKDIR /app

COPY go.mod go.sum stripe_service/stripe-air.toml ./

RUN go install github.com/air-verse/air@latest

RUN go mod download

COPY . .

RUN rm -rf utils

CMD ["air", "-c", "stripe_service/stripe-air.toml"]