FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o swit-auth ./cmd/swit-auth

FROM alpine:latest  

WORKDIR /root/

COPY --from=builder /app/swit-auth .

COPY ./switauth.yaml /root/switauth.yaml

EXPOSE 8081

LABEL org.opencontainers.image.source=https://github.com/innovationmech/swit

ENTRYPOINT ["./swit-auth"]

CMD ["start"] 