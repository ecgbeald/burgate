FROM golang:1.22-alpine

WORKDIR /app

COPY go.mod go.sum kitchen/kitchen-air.toml ./

RUN go install github.com/cosmtrek/air@latest

RUN go mod download

COPY . .

CMD ["air", "-c", "kitchen/kitchen-air.toml"]