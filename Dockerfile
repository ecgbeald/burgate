FROM golang:1.22-alpine

WORKDIR /app

COPY go.mod go.sum gateway-air.toml ./

RUN go install github.com/cosmtrek/air@latest

RUN go mod download

COPY . .

EXPOSE 8888

CMD ["air", "-c", "gateway-air.toml"]