FROM golang:1.15.6-buster

WORKDIR /app
COPY . .

RUN go mod download
RUN go mod verify
RUN go build main.go

ENTRYPOINT ["/app/main"]
