FROM golang:alpine
WORKDIR /build
RUN apk update && apk add --no-cache git
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY ./ .
COPY .env .
RUN CGO_ENABLED=0 go build -a -o main .
EXPOSE 8080
CMD ["./main"]