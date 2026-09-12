package storage

import (
	"context"
	"testing"

	"github.com/wecratfs/commerce/internal/ports"
)

func TestS3RejectsUnscopedUploadKeysBeforeProviderCall(t *testing.T) {
	storage := &S3{}
	if _, err := storage.CreateUploadURL(context.Background(), ports.UploadRequest{ObjectKey: "../private.txt", ContentType: "text/plain", MaxBytes: 10}); err == nil {
		t.Fatal("expected unsafe object key rejection")
	}
}

func TestValidateUploadURL(t *testing.T) {
	if !ValidateUploadURL("https://storage.example/upload") || ValidateUploadURL("javascript:alert(1)") {
		t.Fatal("unexpected upload URL validation result")
	}
}
