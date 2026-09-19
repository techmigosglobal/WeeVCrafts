package main

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestToHTTPRequestPreservesPathQueryBodyAndClientIP(t *testing.T) {
	request, err := toHTTPRequest(events.APIGatewayProxyRequest{
		HTTPMethod: http.MethodPost,
		Path:       "/api/v1/checkout",
		Headers: map[string]string{
			"Host":                      "shop.example.net",
			"X-NF-Client-Connection-IP": "2001:db8::4",
			"Content-Type":              "application/json",
			"X-Forwarded-Proto":         "http",
		},
		MultiValueQueryStringParameters: map[string][]string{"tag": {"craft", "linen"}},
		Body:                            base64.StdEncoding.EncodeToString([]byte(`{"cart_id":7}`)),
		IsBase64Encoded:                 true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.URL.Host != "shop.example.net" || request.URL.Path != "/api/v1/checkout" {
		t.Fatalf("unexpected destination %s", request.URL)
	}
	if values := request.URL.Query()["tag"]; len(values) != 2 || values[0] != "craft" || values[1] != "linen" {
		t.Fatalf("query values were not preserved: %s", request.URL.RawQuery)
	}
	if request.RemoteAddr != "[2001:db8::4]:0" {
		t.Fatalf("trusted Netlify client address not propagated: %q", request.RemoteAddr)
	}
	if request.Header.Get("X-Forwarded-Proto") != "https" {
		t.Fatalf("request scheme header = %q", request.Header.Get("X-Forwarded-Proto"))
	}
	body, err := io.ReadAll(request.Body)
	if err != nil || string(body) != `{"cart_id":7}` {
		t.Fatalf("request body = %q, err=%v", body, err)
	}
}

func TestToHTTPRequestRejectsInvalidBase64(t *testing.T) {
	_, err := toHTTPRequest(events.APIGatewayProxyRequest{HTTPMethod: http.MethodPost, Path: "/", Body: "%%%", IsBase64Encoded: true})
	if err == nil {
		t.Fatal("expected invalid event body to fail")
	}
}

func TestFromHTTPResponsePreservesRepeatedCookiesAndBinaryPayload(t *testing.T) {
	recorder := httptest.NewRecorder()
	recorder.Header().Add("Set-Cookie", "session=a; HttpOnly")
	recorder.Header().Add("Set-Cookie", "csrf=b")
	recorder.Header().Set("Content-Type", "image/webp")
	recorder.WriteHeader(http.StatusOK)
	_, _ = recorder.Write([]byte{0x52, 0x49, 0x46, 0x46})
	response, err := fromHTTPResponse(recorder.Result())
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !response.IsBase64Encoded {
		t.Fatalf("unexpected response metadata: %#v", response)
	}
	cookies := response.MultiValueHeaders["Set-Cookie"]
	if len(cookies) != 2 || !strings.Contains(strings.Join(cookies, ";"), "session=a") || !strings.Contains(strings.Join(cookies, ";"), "csrf=b") {
		t.Fatalf("repeated cookies lost: %v", cookies)
	}
	decoded, err := base64.StdEncoding.DecodeString(response.Body)
	if err != nil || string(decoded) != "RIFF" {
		t.Fatalf("binary response body = %v, err=%v", decoded, err)
	}
}

func TestIsBinaryContentType(t *testing.T) {
	for _, contentType := range []string{"image/avif", "font/woff2", "application/wasm", "application/octet-stream"} {
		if !isBinaryContentType(contentType) {
			t.Errorf("%q should be base64 encoded", contentType)
		}
	}
	if isBinaryContentType("text/html; charset=utf-8") {
		t.Fatal("HTML should remain text")
	}
}
