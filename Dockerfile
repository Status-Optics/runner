FROM golang:1.24
RUN apt-get update && apt-get install -y git python3

WORKDIR /app
COPY . .
RUN go mod download
COPY . .
RUN chmod +x generate-sha.sh && ./generate-sha.sh
RUN go build -o main .


CMD ["/app/main"]
# Use the following command to build the Docker image
# docker build -t my-go-app .
# Use the following command to run the Docker container
# docker run -p 8080:8080 --env-file .env my-go-app
