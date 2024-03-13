# Stage 1: Build the application and dependencies
# 这个是 github action 用的
FROM golang:1.21

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files
COPY webook /app/webook

# Set the Env variable
ENV EGO_DEBUG=true

# Set the command to run when the container starts
CMD ["/app/webook", "--config=config/local.yaml"]