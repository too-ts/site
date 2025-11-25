FROM golang:1.25.3 as SITE

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/*.go ./
COPY views/ ./views/
COPY css/ ./css/

RUN go build main.go 

EXPOSE 443

CMD ["./main"]
