# Use a base image with Go installed
FROM golang:1.22.2-bullseye as build

# Set the working directory
WORKDIR /app

# Copy the Go source code
COPY go.mod go.sum ./
COPY main.go ./
COPY ./bootstrap ./

RUN apt-get update && apt-get install -y --no-install-recommends \
    libvips-dev zip \
    && rm -rf /var/lib/apt/lists/*

# Build the Go binary
RUN GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o main main.go

# Create a directory to hold the shared libraries
RUN mkdir image-compressor

# Copy each libvips dependency into application root
RUN root_val="/usr"
RUN DEPENDENCIES=$(ldd /usr/lib/x86_64-linux-gnu/libvips.so.42 | awk '/=>/ {print $3}') \
    && for dep in $DEPENDENCIES; do \
        depen_loc=$root_val$dep; \
        cp $depen_loc ./image-compressor; \
    done

# Copy libvips into application root
RUN cp /usr/lib/x86_64-linux-gnu/libvips.so.42 ./image-compressor

# Create the directory structure for the ZIP archive
RUN mv main image-compressor/
RUN mv bootstrap image-compressor/

# # Change the working directory to the parent directory
WORKDIR /app/image-compressor

# # Create the ZIP archive
RUN zip -r image-compressor.zip .

RUN chmod +x /app/image-compressor/image-compressor.zip

# Use a minimal base image for the Lambda function
FROM public.ecr.aws/lambda/provided:latest

# Ensure execution
RUN chmod +x /usr/local/bin/aws-lambda-rie

# Copy the ZIP archive from the build stage
COPY --from=build /app/image-compressor/image-compressor.zip .

# Set the command to run the Lambda function
CMD ["./bootstrap"]