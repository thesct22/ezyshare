FROM golang:1.16-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY *.go ./
COPY ./server ./server


RUN go build -o /backend-go

CMD [ "/backend-go" ]
