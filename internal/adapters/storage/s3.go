package storage

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/wecratfs/commerce/internal/ports"
)

type S3 struct {
	client *minio.Client
	bucket string
}

func NewS3(endpoint, accessKey, secretKey, bucket string, secure bool) (*S3, error) {
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")
	client, err := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(accessKey, secretKey, ""), Secure: secure})
	if err != nil {
		return nil, err
	}
	return &S3{client: client, bucket: bucket}, nil
}

func (s *S3) CreateUploadURL(ctx context.Context, request ports.UploadRequest) (ports.UploadURL, error) {
	if s.client == nil || s.bucket == "" || !validProductObjectKey(request.ObjectKey) || request.MaxBytes <= 0 || request.MaxBytes > 25<<20 {
		return ports.UploadURL{}, errors.New("invalid upload request")
	}
	presigned, err := s.client.PresignedPutObject(ctx, s.bucket, request.ObjectKey, 15*time.Minute)
	if err != nil {
		return ports.UploadURL{}, err
	}
	return ports.UploadURL{URL: presigned.String(), ExpiresAtUnix: time.Now().Add(15 * time.Minute).Unix()}, nil
}

func (s *S3) InspectObject(ctx context.Context, objectKey string) (ports.ObjectInfo, error) {
	if s.client == nil || s.bucket == "" || !validProductObjectKey(objectKey) {
		return ports.ObjectInfo{}, errors.New("invalid object key")
	}
	info, err := s.client.StatObject(ctx, s.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return ports.ObjectInfo{}, err
	}
	return ports.ObjectInfo{ContentType: info.ContentType, ByteSize: info.Size}, nil
}

func (s *S3) CreateDownloadURL(ctx context.Context, objectKey string, lifetime time.Duration) (ports.DownloadURL, error) {
	if s.client == nil || s.bucket == "" || !validProductObjectKey(objectKey) || lifetime <= 0 || lifetime > 24*time.Hour {
		return ports.DownloadURL{}, errors.New("invalid download request")
	}
	presigned, err := s.client.PresignedGetObject(ctx, s.bucket, objectKey, lifetime, nil)
	if err != nil {
		return ports.DownloadURL{}, err
	}
	return ports.DownloadURL{URL: presigned.String(), ExpiresAtUnix: time.Now().Add(lifetime).Unix()}, nil
}

func (s *S3) DeleteObject(ctx context.Context, objectKey string) error {
	if s.client == nil || s.bucket == "" || !validProductObjectKey(objectKey) {
		return errors.New("invalid object key")
	}
	return s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}

func (s *S3) EnsureBucket(ctx context.Context) error {
	if s.client == nil || s.bucket == "" {
		return errors.New("storage bucket is not configured")
	}
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
}

func (s *S3) VerifyBucket(ctx context.Context) error {
	if s.client == nil || s.bucket == "" {
		return errors.New("storage bucket is not configured")
	}
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("storage bucket does not exist")
	}
	return nil
}

func ValidateUploadURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func validProductObjectKey(value string) bool {
	return strings.HasPrefix(value, "products/") &&
		!strings.Contains(value, "..") &&
		!strings.HasPrefix(value, "/") &&
		!strings.ContainsAny(value, "\\\n\r\t")
}
