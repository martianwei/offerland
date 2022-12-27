FROM golang:1.19

WORKDIR /app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -ldflags='-s' -o=./bin/api ./cmd/api

# ADD ENV

EXPOSE 8080

CMD make run/api