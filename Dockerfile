FROM golang:1.21 as build
WORKDIR /image-compressor
# Copy dependencies list
COPY go.mod go.sum ./
# Build with optional lambda.norpc tag
COPY main.go .
# Set PKG_CONFIG_PATH to include libvips directory

# gets past build state at least...
RUN apt-get update && apt-get install -y --no-install-recommends \
    libvips libvips-dev \
    && rm -rf /var/lib/apt/lists/*

RUN ldconfig

# new attempt - BUSTED 
# RUN apt-get update -qq && apt-get install -y --no-install-recommends libvips42


ENV PKG_CONFIG_PATH /usr/lib/x86_64-linux-gnu/pkgconfig
ENV LD_LIBRARY_PATH=/usr/lib/x86_64-linux-gnu:$LD_LIBRARY_PATH
# Update dynamic linker's cache

RUN go build -tags lambda.norpc -o main main.go
# Copy artifacts to a clean image
FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build /image-compressor/main ./main
ENTRYPOINT [ "./main" ]
