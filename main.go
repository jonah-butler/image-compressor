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

const TEMP_UPLOADS_DIR = "temp_uploads"

// typing ease helper
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

type LambdaPayload struct {
	UploadPath string
	UploadTags string
	Upload     []byte
}

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

		} else {

			content, err := io.ReadAll(part)
			if err != nil {
				return nil, err
			}

			if part.FormName() == "upload-path" {
				lp.UploadPath = string(content)
			} else if part.FormName() == "upload-tags" {
				lp.UploadTags = string(content)
			}

		}
	}

	return &lp, nil
}

func createFolder(dirname string) error {
	_, err := os.Stat(dirname)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(dirname, 0755)
		if errDir != nil {
			return errDir
		}
	}
	return nil
}

func compressImage(buffer []byte, filename string) ([]byte, error) {
	processed, err := bimg.NewImage(buffer).Process(bimg.Options{
		StripMetadata: true,
		Palette:       true,
		// Speed: 9, / /may not need this...
	})
	if err != nil {
		return nil, err
	}

	return processed, nil
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	currentDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	os.Setenv("LD_LIBRARY_PATH", currentDir)

	headers := convertToStandardHeader(request.Headers)

	contentType := headers.Get("content-type")

	reader, err := decodeRequestBody(request, contentType)
	if err != nil {
		return errorHandler(err)
	}

	payload, err := extractFormData(reader)
	if err != nil {
		return errorHandler(err)
	}
	log.Println("payload: ", payload)

	// errDir := createFolder("test_uploads")
	// if errDir != nil {
	// 	panic(errDir)
	// }

	imageData, err := compressImage(payload.Upload, "test1.png")
	if err != nil {
		return errorHandler(err)
	}
	log.Println("got the image data!")

	// img, _, err := image.Decode(bytes.NewReader(imageData))
	// if err != nil {
	// 	return errorHandler(err)
	// }

	// log.Println("decode the image!")

	// var buf bytes.Buffer
	// if err := png.Encode(&buf, img); err != nil {
	// 	return errorHandler(err)
	// }

	// log.Println("encoded the image!")

	// Set the headers
	responseHeaders := map[string]string{
		"Content-Type": "image/png",
	}

	// Encode the image data to base64
	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	log.Println("encoded to base64!")

	// return events.APIGatewayProxyResponse{
	// 	StatusCode: 200,
	// 	Body:       "hmmmmmmmm",
	// }, nil

	// Return the response
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
