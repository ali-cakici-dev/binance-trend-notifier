FROM golang:alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
RUN apk add --no-cache curl

COPY . .

CMD ["go", "run", "cmd/main.go"]
