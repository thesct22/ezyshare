FROM golang:1.16

COPY . /src

WORKDIR /src

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o backend-go .



FROM heroku/heroku:18

WORKDIR /app

COPY --from=0 /src/backend-go /app

CMD ["./backend-go"]