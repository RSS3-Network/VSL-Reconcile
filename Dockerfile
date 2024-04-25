FROM golang:alpine AS Builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod files only
COPY go.mod .
#COPY go.sum .

# Download all the dependencies
RUN go mod download

# Copy everything from the current directory to the PWD (Present Working Directory) inside the container
COPY . .

# Build image
RUN go build -o app .

FROM scratch AS Runner

WORKDIR /app

COPY --from=Builder /app/app /app/app

VOLUME /app/data

ENV MODE=prod

# Run the executable
CMD ["/app/app"]
