FROM golang:1.21 as build
WORKDIR /image-compressor
COPY go.mod go.sum ./
COPY main.go .

RUN apt-get update && apt-get install -y --no-install-recommends \
    libvips-dev \
    && rm -rf /var/lib/apt/lists/*

ENV PKG_CONFIG_PATH /usr/lib/x86_64-linux-gnu/pkgconfig
ENV LD_LIBRARY_PATH=/usr/lib/x86_64-linux-gnu:$LD_LIBRARY_PATH

RUN ldconfig

RUN go build -tags lambda.norpc -o main main.go
FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build /image-compressor/main ./main
ENTRYPOINT [ "./main" ]
