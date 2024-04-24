package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/h2non/bimg"
)

const MAX_LAMBDA_RETURN_SIZE = 6 // in mb
const NUM_OF_BYTES_IN_MB = 1048576.0

type LambdaPayload struct {
	Upload []byte // <upload>
}

func errorHandler(err error) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{StatusCode: 400, Body: err.Error()}, err
}

// simplifies getting content-type or other header data
func convertToStandardHeader(header map[string]string) http.Header {
	h := http.Header{}
	for k, v := range header {
		h.Add(strings.TrimSpace(k), v)
	}
	return h
}

/*
Manual parsing of multipart/form-data fields since .ParseForm isn't available.
-> validate type of body, ensure it includes multipart fields
-> ensure header data is accurate and trimmed
-> extract boundary identifier
-> if b64 encoded, decode body first
-> return decoded multipart.Reader
*/
func decodeRequestBody(request events.APIGatewayProxyRequest, contentType string) (*multipart.Reader, error) {
	mediatype, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}

	// ensure mediaType is multipart
	if strings.Index(strings.ToLower(strings.TrimSpace(mediatype)), "multipart/") != 0 {
		return nil, errors.New("content type invalid")
	}

	paramKeys := convertToStandardHeader(params)

	boundary := paramKeys.Get("boundary")
	if len(boundary) == 0 {
		return nil, err
	}

	if request.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(request.Body)
		if err != nil {
			return nil, err
		}
		return multipart.NewReader(bytes.NewReader(decoded), boundary), nil
	}
	return multipart.NewReader(strings.NewReader(request.Body), boundary), nil

}

// Max return size of a lambda payload is 6mb.
// Returns boolean based on newly compressed size to determine if response will be successful.
func isAcceptableSizeLimit(imageSize int) bool {
	return (float64(imageSize) / NUM_OF_BYTES_IN_MB) <= MAX_LAMBDA_RETURN_SIZE
}

// Parses form-data fields
func extractFormData(reader *multipart.Reader) (*LambdaPayload, error) {
	lp := LambdaPayload{}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if part.FileName() != "" && part.FormName() == "upload" {

			file, err := io.ReadAll(part)
			if err != nil {
				return nil, err
			}
			lp.Upload = file

		}
	}

	return &lp, nil
}

func compressImage(buffer []byte) ([]byte, error) {
	processed, err := bimg.NewImage(buffer).Process(bimg.Options{
		StripMetadata: true,
		Palette:       true,
	})
	if err != nil {
		return nil, err
	}

	return processed, nil
}

func validateFileMime(buffer []byte) bool {
	acceptedMimeTypes := []string{"image/png", "image/jpeg"} // just two for now

	mime := http.DetectContentType(buffer)

	isValid := false

	for _, mimeType := range acceptedMimeTypes {
		if mimeType == mime {
			isValid = true
			break
		}
	}

	return isValid
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	currentDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Println("failed to retrieve file path")
		panic(err)
	}
	os.Setenv("LD_LIBRARY_PATH", currentDir)

	headers := convertToStandardHeader(request.Headers)

	contentType := headers.Get("content-type")

	reader, err := decodeRequestBody(request, contentType)
	if err != nil {
		log.Println("failed to decode request body")
		return errorHandler(err)
	}

	payload, err := extractFormData(reader)
	if err != nil {
		log.Println("failed to extract form data")
		return errorHandler(err)
	}

	isValidMime := validateFileMime(payload.Upload)
	if !isValidMime {
		log.Println("upload is invalid type")
		return errorHandler(errors.New("invalid file type"))
	}

	imageData, err := compressImage(payload.Upload)
	if err != nil {
		log.Println("failed to compress image")
		return errorHandler(err)
	}

	sizeIsOk := isAcceptableSizeLimit(len(imageData))
	if !sizeIsOk {
		log.Println("compressed image exceeds 6mb")
		return errorHandler(errors.New("image size exceeded - can not process request"))
	}

	responseHeaders := map[string]string{
		"Content-Type": http.DetectContentType(imageData),
	}

	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	return events.APIGatewayProxyResponse{
		StatusCode:      http.StatusOK,
		Headers:         responseHeaders,
		Body:            imageBase64,
		IsBase64Encoded: true,
	}, nil
}

func main() {
	lambda.Start(handler)
}
