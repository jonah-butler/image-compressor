# START OLD
# FROM golang:1.21 as build
# WORKDIR /image-compressor
# COPY go.mod go.sum ./
# COPY main.go .

# RUN apt-get update && apt-get install -y --no-install-recommends \
#     libvips-dev \
#     && rm -rf /var/lib/apt/lists/*

# # apparently do not need this
# # ENV PKG_CONFIG_PATH /usr/lib/x86_64-linux-gnu/pkgconfig
# # ENV LD_LIBRARY_PATH=/usr/lib/x86_64-linux-gnu:$LD_LIBRARY_PATH

# # RUN ldconfig

# # ENV LD_LIBRARY_PATH /usr/local/lib:$LD_LIBRARY_PATH


# # Download and compile libvips from source
# # RUN curl -sLO https://github.com/libvips/libvips/releases/download/v8.12.2/vips-8.12.2.tar.gz \
# #     && tar -xzf vips-8.12.2.tar.gz \
# #     && cd vips-8.12.2 \
# #     && ./configure \
# #     && make \
# #     && make install \
# #     && ldconfig

# RUN go build -tags lambda.norpc -o main main.go

# FROM public.ecr.aws/lambda/provided:al2023

# COPY --from=build /usr/lib/x86_64-linux-gnu/* ./
# COPY --from=build /image-compressor/main ./main
# ENTRYPOINT [ "./main" ]
# END OLD

# Use a base image with Go installed
FROM golang:1.21 as build

# Set the working directory
WORKDIR /app

# Copy the Go source code
COPY go.mod go.sum ./
COPY main.go ./
COPY ./bootstrap ./

RUN apt-get update && apt-get install -y --no-install-recommends \
    libvips-dev zip\
    && rm -rf /var/lib/apt/lists/*

# # Build the Go binary
RUN GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o main main.go

# Create a directory to hold the shared libraries
RUN mkdir myapp

RUN root_val="/usr"
RUN DEPENDENCIES=$(ldd /usr/lib/x86_64-linux-gnu/libvips.so.42 | awk '/=>/ {print $3}') \
    && for dep in $DEPENDENCIES; do \
        depen_loc=$root_val$dep; \
        cp $depen_loc ./myapp; \
    done
RUN cp /usr/lib/x86_64-linux-gnu/libvips.so.42 ./myapp
# # Copy the shared libraries into the libs directory
# RUN cp -r /usr/lib/x86_64-linux-gnu/* ./libs/
# # COPY --from=build /usr/lib/x86_64-linux-gnu/* ./libs/
# # You may need to adjust the path depending on the location of the libraries

# Create the directory structure for the ZIP archive
RUN mv main myapp/
RUN mv bootstrap myapp/
# RUN mv libs myapp/

# # Change the working directory to the parent directory
WORKDIR /app/myapp

# # Create the ZIP archive
RUN zip -r myapp.zip .

# Use a minimal base image for the Lambda function
# FROM public.ecr.aws/lambda/provided:al2023
FROM amazonlinux:2

# # Install any necessary dependencies
RUN yum install -y curl

# Download the AWS Lambda RIE binary
RUN curl -Lo /usr/local/bin/aws-lambda-rie https://github.com/aws/aws-lambda-runtime-interface-emulator/releases/latest/download/aws-lambda-rie

# Make the binary executable
RUN chmod +x /usr/local/bin/aws-lambda-rie

# Copy the ZIP archive from the build stage
COPY --from=build /app/myapp/myapp.zip .

# Set the command to run the Lambda function
CMD ["./main"]
