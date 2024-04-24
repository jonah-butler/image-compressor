# Golang Image Compression Lambda

## Overview

> This project includes the necessary Go files and generation scripts to build a simple Lambda function for compressing images with [bimg](https://github.com/h2non/bimg)

## Includes

- Dockerfile for generating the `.zip` to be uploaded to AWS Lambda
- main.go for handling form-data file uploads

## Build Steps

- `docker build --platform linux/amd64 -t docker-image_compressor:compressor .`
- `docker run -it {container_id} /bin/bash`
- `docker cp {image_id}:/var/task/image-compressor.zip ~/{some_location_on_machine}`
- upload newly created `.zip` to AWS Lambda using:
  - x86_64 architecture
  - Amazon Linux 2023 runtime

## Payload

**request method:** `POST`

**body:**: `form-data`

**accepted_key_value_pair:**

- `upload`: `<File>`

**response:** `b64 image data`
