FROM golang:1.22

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o api ./cmd/api/main.go
RUN go build -o worker ./cmd/worker/main.go

CMD ["./api"]
