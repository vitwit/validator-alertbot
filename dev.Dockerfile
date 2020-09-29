FROM golang:1.13.1

COPY . /app/validator-alertbot

WORKDIR /app/validator-alertbot

RUN go mod download

CMD ["go", "run", "main.go"]