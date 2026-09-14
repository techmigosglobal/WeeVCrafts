package main

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestHandlerSmoke(t *testing.T) {
	// 1. Home page renders HTML.
	home, err := handler(events.APIGatewayProxyRequest{HTTPMethod: "GET", Path: "/"})
	if err != nil {
		t.Fatalf("home: %v", err)
	}
	if home.StatusCode != 200 {
		t.Fatalf("home status = %d", home.StatusCode)
	}
	if !strings.Contains(home.Body, "WeeVCrafts") {
		t.Fatal("home body missing wordmark")
	}
	if home.IsBase64Encoded {
		t.Fatal("home must not be base64")
	}

	// 2. Binary asset is base64-encoded with image content type.
	asset, err := handler(events.APIGatewayProxyRequest{HTTPMethod: "GET", Path: "/assets/images/customer/saree-maroon.webp"})
	if err != nil {
		t.Fatalf("asset: %v", err)
	}
	if asset.StatusCode != 200 {
		t.Fatalf("asset status = %d", asset.StatusCode)
	}
	if !asset.IsBase64Encoded {
		t.Fatal("webp asset must be base64")
	}
	if ct := strings.Join(asset.MultiValueHeaders["Content-Type"], ","); ct != "image/webp" {
		t.Fatalf("asset content type = %q", ct)
	}
	if _, err := base64.StdEncoding.DecodeString(asset.Body); err != nil {
		t.Fatalf("asset body is not valid base64: %v", err)
	}

	// 3. Session cookie is set via MultiValueHeaders Set-Cookie.
	cookie := strings.Join(home.MultiValueHeaders["Set-Cookie"], "; ")
	if cookie == "" || !strings.Contains(cookie, "weevcrafts_ui=") {
		t.Fatalf("session cookie missing, got %q", cookie)
	}

	// 4. HTMX cart mutation POST works (HX-Request header like htmx sends).
	mut, err := handler(events.APIGatewayProxyRequest{
		HTTPMethod: "POST",
		Path:       "/ui/cart/madhubani-tree",
		Headers: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
			"HX-Request":   "true",
		},
		Body: "delta=1",
	})
	if err != nil {
		t.Fatalf("mutation: %v", err)
	}
	if mut.StatusCode != 200 {
		t.Fatalf("mutation status = %d", mut.StatusCode)
	}

	// 5. Unknown route renders the preview 404 page.
	missing, err := handler(events.APIGatewayProxyRequest{HTTPMethod: "GET", Path: "/no-such-page"})
	if err != nil {
		t.Fatalf("404 route: %v", err)
	}
	if missing.StatusCode != 404 {
		t.Fatalf("404 route status = %d", missing.StatusCode)
	}

	// 6. Search query params round-trip.
	search, err := handler(events.APIGatewayProxyRequest{
		HTTPMethod:                      "GET",
		Path:                            "/search",
		MultiValueQueryStringParameters: map[string][]string{"q": {"saree"}},
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if search.StatusCode != 200 || !strings.Contains(search.Body, "saree") {
		t.Fatalf("search did not apply query: status=%d", search.StatusCode)
	}
}
