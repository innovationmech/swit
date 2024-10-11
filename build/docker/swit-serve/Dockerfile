FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o switserve ./cmd/switserve

FROM alpine:latest  

WORKDIR /root/

COPY --from=builder /app/swit-serve .

COPY ./swit.yaml /root/swit.yaml

EXPOSE 8080

LABEL org.opencontainers.image.source=https://github.com/innovationmech/swit

ENTRYPOINT ["./swit-serve"]

CMD ["serve"]
