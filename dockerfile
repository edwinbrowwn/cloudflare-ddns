FROM golang:1.16-alpine

WORKDIR /

COPY go.mod ./
# COPY go.sum ./
RUN go mod download

COPY "config.json" "./"
COPY *.go ./

RUN go build -o /cloudflare-ddns

EXPOSE 8080

CMD [ "/cloudflare-ddns" ]