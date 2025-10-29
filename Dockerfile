FROM golang:1.23.4-bullseye

WORKDIR /app
COPY . .

ARG GIT_COMMIT
ENV GIT_COMMIT $GIT_COMMIT

RUN go mod download
RUN go mod verify
RUN go build main.go

ENTRYPOINT ["/app/main"]
