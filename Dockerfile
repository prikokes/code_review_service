FROM golang:1.25.1-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . ./

RUN go build -o /pr-reviewer ./cmd/main.go

EXPOSE 8080

CMD [ "/pr-reviewer" ]
