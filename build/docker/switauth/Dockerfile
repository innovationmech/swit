FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o switauth ./cmd/swit-auth

FROM alpine:latest  

WORKDIR /root/

COPY --from=builder /app/switauth .

COPY ./switauth.yaml /root/switauth.yaml

EXPOSE 8081

LABEL org.opencontainers.image.source=https://github.com/innovationmech/swit

ENTRYPOINT ["./switauth"]

CMD ["start"] 