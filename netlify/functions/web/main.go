// Package main adapts Netlify's Lambda-compatible request event to the real
// PostgreSQL-backed Go application. It does not use the isolated UI preview.
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/wecratfs/commerce/internal/apphost"
)

var runtimeMu sync.Mutex
var applicationRuntime *apphost.Runtime

func getRuntime() (*apphost.Runtime, error) {
	runtimeMu.Lock()
	defer runtimeMu.Unlock()
	if applicationRuntime != nil {
		return applicationRuntime, nil
	}
	initialized, err := apphost.NewNetlify(slog.Default())
	if err != nil {
		return nil, err
	}
	applicationRuntime = initialized
	return applicationRuntime, nil
}

// toHTTPRequest rebuilds an http.Request from the Lambda event. The event
// path retains the routed request path, and query strings are re-encoded so
// that r.URL.Query() behaves as it does on a conventional HTTP server.
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
	path := event.Path
	if path == "" {
		path = "/"
	}
	rawURL := path
	if query != "" {
		rawURL += "?" + query
	}
	body := []byte(event.Body)
	if event.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(event.Body)
		if err != nil {
			return nil, err
		}
		body = decoded
	}
	method := event.HTTPMethod
	if method == "" {
		method = http.MethodGet
	}
	host := headerValue(event.Headers, "host")
	if host == "" {
		host = "weevcrafts.netlify.app"
	}
	request, err := http.NewRequest(method, "https://"+host+rawURL, bytes.NewReader(body))
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
	request.Header.Set("X-Forwarded-Proto", "https")
	if clientIP := headerValue(event.Headers, "x-nf-client-connection-ip"); clientIP != "" {
		request.RemoteAddr = remoteAddress(clientIP)
	}
	if len(body) > 0 {
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

func headerValue(headers map[string]string, name string) string {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func remoteAddress(clientIP string) string {
	clientIP = strings.TrimSpace(strings.Trim(clientIP, "[]"))
	if parsed := net.ParseIP(clientIP); parsed != nil {
		return net.JoinHostPort(parsed.String(), "0")
	}
	return ""
}

// fromHTTPResponse converts the handler response into the Lambda proxy shape.
// Binary payloads are base64-encoded; repeated Set-Cookie headers are kept.
func fromHTTPResponse(response *http.Response) (events.APIGatewayProxyResponse, error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	contentType := response.Header.Get("Content-Type")
	return events.APIGatewayProxyResponse{
		StatusCode:        response.StatusCode,
		MultiValueHeaders: response.Header,
		Body:              encodeBody(body, contentType),
		IsBase64Encoded:   isBinaryContentType(contentType),
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
	for _, prefix := range []string{"image/", "font/", "audio/", "video/", "application/octet-stream", "application/wasm"} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func handler(event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	runtime, err := getRuntime()
	if err != nil {
		slog.Error("initialize Netlify application runtime", "error", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusServiceUnavailable,
			Headers:    map[string]string{"Content-Type": "text/plain; charset=utf-8", "Cache-Control": "no-store"},
			Body:       "WeeVCrafts backend is not configured or is temporarily unavailable.",
		}, nil
	}
	request, err := toHTTPRequest(event)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}, nil
	}
	requestContext, cancel := context.WithTimeout(request.Context(), 25*time.Second)
	defer cancel()
	request = request.WithContext(requestContext)
	recorder := httptest.NewRecorder()
	runtime.Handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	defer response.Body.Close()
	return fromHTTPResponse(response)
}

func main() {
	lambda.Start(handler)
}
