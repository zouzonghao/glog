# Use a lightweight Alpine image for the final production environment.
FROM alpine:latest

# Install runtime dependencies.
# ca-certificates is for HTTPS connections.
# tzdata is for time zone support.
RUN apk add --no-cache ca-certificates tzdata

# Set the time zone
ENV TZ=Asia/Shanghai

# Set the working directory
WORKDIR /app

# Copy the pre-built binary from the build context.
# The CI job must have already built 'glog-linux-amd64'.
COPY glog-linux-amd64 /app/glog

# Make the binary executable
RUN chmod +x /app/glog

# Expose the port the application runs on
EXPOSE 37371

# Define the command to run the application
CMD ["./glog"]
