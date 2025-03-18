FROM golang:1.24-alpine
RUN apk update && apk add --no-cache git python3 py3-pip

WORKDIR /app
COPY . .
# RUN chmod +x generate-sha.sh && ./generate-sha.sh
RUN go mod download
RUN go build -o main .

CMD ["/app/main"]
# Use the following command to build the Docker image
# docker build -t my-go-app .
# Use the following command to run the Docker container
# docker run -p 8080:8080 --env-file .env my-go-app
