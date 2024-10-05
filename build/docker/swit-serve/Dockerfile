FROM golang:1.20-alpine AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o swit-serve ./cmd/swit-serve

FROM alpine:latest  

WORKDIR /root/

COPY --from=builder /app/swit-serve .

COPY ./swit.yaml /root/swit.yaml

EXPOSE 8080

ENTRYPOINT ["./swit-serve"]

CMD ["serve"]
