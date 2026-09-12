package commerce

import "time"

type MediaUpload struct {
	ID            int64  `json:"id"`
	ProductID     int64  `json:"product_id"`
	ObjectKey     string `json:"object_key"`
	ContentType   string `json:"content_type"`
	MaxBytes      int64  `json:"max_bytes"`
	UploadURL     string `json:"upload_url"`
	ExpiresAtUnix int64  `json:"expires_at_unix"`
}

type MediaRecord struct {
	ID          int64     `json:"id"`
	ProductID   int64     `json:"product_id"`
	SellerID    int64     `json:"seller_id"`
	ObjectKey   string    `json:"object_key"`
	ContentType string    `json:"content_type"`
	ByteSize    int64     `json:"byte_size"`
	Visibility  string    `json:"visibility"`
	Status      string    `json:"status"`
	PublicPath  string    `json:"public_path"`
	CreatedAt   time.Time `json:"created_at"`
}
