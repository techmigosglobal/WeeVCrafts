// Netlify Functions adapter for the WeeVCrafts UI preview.
//
// Netlify Functions support Go through the Lambda-compatible API, so this
// package converts API Gateway-style events into net/http requests and runs
// the unmodified storefront preview handler (see cmd/web). The preview store
// is an in-memory fixture behind a per-browser cookie, which is exactly what
// a single warm Lambda execution environment provides.
//
// The function is compiled by the netlify.toml build command into
// netlify/functions-dist/web, so this source directory is documentation for
// the adapter; the deployed artifact is the pre-built binary of the same name.
package main

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/wecratfs/commerce/internal/web/handlers"
	"github.com/wecratfs/commerce/internal/web/middleware"
)

var preview http.Handler

func init() {
	preview = middleware.PreviewHeaders(handlers.NewMockHandler())
}

// toHTTPRequest rebuilds an http.Request from the Lambda event. The event
// path already contains the original URL-encoded path, and query strings
// must be re-encoded so that r.URL.Query() behaves like a direct request.
func toHTTPRequest(event events.APIGatewayProxyRequest) (*http.Request, error) {
	query := ""
	if len(event.MultiValueQueryStringParameters) > 0 {
		query = encodeMultiValues(event.MultiValueQueryStringParameters)
	} else if len(event.QueryStringParameters) > 0 {
		values := url.Values{}
		for key, value := range event.QueryStringParameters {
			values.Set(key, value)
		}
		query = values.Encode()
	}
	rawURL := event.Path
	if query != "" {
		rawURL += "?" + query
	}
	var body string
	if event.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(event.Body)
		if err != nil {
			return nil, err
		}
		body = string(decoded)
	} else {
		body = event.Body
	}
	request, err := http.NewRequest(event.HTTPMethod, "http://preview.local"+rawURL, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.RequestURI = rawURL
	for key, values := range event.MultiValueHeaders {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}
	for key, value := range event.Headers {
		if request.Header.Get(key) == "" {
			request.Header.Set(key, value)
		}
	}
	if request.Header.Get("X-Forwarded-Proto") == "" {
		request.Header.Set("X-Forwarded-Proto", "https")
	}
	if event.Body != "" {
		request.ContentLength = int64(len(body))
	}
	return request, nil
}

func encodeMultiValues(values map[string][]string) string {
	result := url.Values{}
	for key, list := range values {
		for _, value := range list {
			result.Add(key, value)
		}
	}
	return result.Encode()
}

// fromHTTPResponse converts the handler response into the Lambda proxy
// shape. Binary payloads (embedded images, fonts) must be base64-encoded,
// and Set-Cookie repeats via MultiValueHeaders.
func fromHTTPResponse(response *http.Response) (events.APIGatewayProxyResponse, error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	return events.APIGatewayProxyResponse{
		StatusCode:        response.StatusCode,
		MultiValueHeaders: response.Header,
		Body:              encodeBody(body, response.Header.Get("Content-Type")),
		IsBase64Encoded:   isBinaryContentType(response.Header.Get("Content-Type")),
	}, nil
}

func encodeBody(body []byte, contentType string) string {
	if isBinaryContentType(contentType) {
		return base64.StdEncoding.EncodeToString(body)
	}
	return string(body)
}

func isBinaryContentType(contentType string) bool {
	value := strings.ToLower(contentType)
	for _, prefix := range []string{"image/", "font/", "audio/", "video/", "application/octet-stream"} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func handler(event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	request, err := toHTTPRequest(event)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
	}
	recorder := httptest.NewRecorder()
	preview.ServeHTTP(recorder, request)
	response := recorder.Result()
	defer response.Body.Close()
	return fromHTTPResponse(response)
}

func main() {
	lambda.Start(handler)
}
