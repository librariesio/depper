FROM golang:1.21.1-bullseye

WORKDIR /app
COPY . .

ARG GIT_COMMIT
ENV GIT_COMMIT $GIT_COMMIT

ARG BUGSNAG_API_KEY
ENV BUGSNAG_API_KEY $BUGSNAG_API_KEY

RUN go mod download
RUN go mod verify
RUN go build main.go

ENTRYPOINT ["/app/main"]
