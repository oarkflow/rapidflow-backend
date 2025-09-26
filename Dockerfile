FROM golang:1.21 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o docker-app .

FROM ubuntu:latest
RUN apt-get update && apt-get install -y ca-certificates
COPY --from=builder /app/docker-app /usr/local/bin/
EXPOSE 3000
CMD ["docker-app", "server"]
