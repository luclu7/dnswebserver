FROM golang:1.15

WORKDIR /src

COPY . .

# Install the package
RUN go build

# This container exposes port 8080 to the outside world
EXPOSE 8080

# Run the executable
CMD ["./dnswebserver"]
