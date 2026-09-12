package postgres

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	storageadapter "github.com/wecratfs/commerce/internal/adapters/storage"
	applicationcommerce "github.com/wecratfs/commerce/internal/application/commerce"
)

func TestMediaUploadFinalizeAndPublicReadAgainstPostgresAndMinIO(t *testing.T) {
	if os.Getenv("S3_INTEGRATION") != "1" {
		t.Skip("S3_INTEGRATION=1 is required for PostgreSQL and MinIO media integration")
	}
	pool := openIntegrationPool(t)
	fixture := newCommerceFixture(t, pool, 1, 1)
	storage, err := storageadapter.NewS3(envOrDefault("S3_ADDR", "localhost:9000"), envOrDefault("S3_ACCESS_KEY", "wecratfs"), envOrDefault("S3_SECRET_KEY", "wecratfs-local-only"), envOrDefault("S3_BUCKET", "wecratfs-media"), false)
	if err != nil {
		t.Fatalf("create storage adapter: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := storage.EnsureBucket(ctx); err != nil {
		t.Fatalf("ensure storage bucket: %v", err)
	}
	repository := NewMediaRepository(pool)
	service := applicationcommerce.NewMediaService(storage, repository)
	payload := []byte("real local media bytes")
	upload, err := service.PrepareUpload(ctx, fixture.ownerID, fixture.productID, "image/png", int64(len(payload)))
	if err != nil {
		t.Fatalf("prepare upload: %v", err)
	}
	t.Cleanup(func() { _ = storage.DeleteObject(context.Background(), upload.ObjectKey) })
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, upload.UploadURL, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("create upload request: %v", err)
	}
	request.Header.Set("Content-Type", "image/png")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("put presigned media: %v", err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("presigned upload returned status %d", response.StatusCode)
	}
	ready, err := service.FinalizeUpload(ctx, fixture.ownerID, upload.ID)
	if err != nil {
		t.Fatalf("finalize uploaded media: %v", err)
	}
	if ready.Status != "ready" || ready.PublicPath != "/media/products/"+strconv.FormatInt(upload.ID, 10) {
		t.Fatalf("unexpected finalized media: %+v", ready)
	}
	publicURL, err := service.ResolvePublicURL(ctx, upload.ID)
	if err != nil {
		t.Fatalf("resolve public media: %v", err)
	}
	publicResponse, err := http.Get(publicURL)
	if err != nil {
		t.Fatalf("read public media: %v", err)
	}
	readBack, readErr := io.ReadAll(publicResponse.Body)
	_ = publicResponse.Body.Close()
	if readErr != nil || publicResponse.StatusCode != http.StatusOK || !bytes.Equal(readBack, payload) {
		t.Fatalf("public media read failed: status=%d err=%v body=%q", publicResponse.StatusCode, readErr, readBack)
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
