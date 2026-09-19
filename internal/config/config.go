package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Address            string
	PublicURL          string
	DatabaseURL        string
	DatabaseMaxConns   int32
	RedisAddress       string
	MeiliAddress       string
	MeiliAPIKey        string
	S3Address          string
	SecureCookies      bool
	SMTPAddress        string
	SMTPUsername       string
	SMTPPassword       string
	SMTPFrom           string
	S3AccessKey        string
	S3SecretKey        string
	S3Bucket           string
	S3Secure           bool
	S3AutoCreateBucket bool
}

func FromEnv() Config {
	return fromEnv(strings.EqualFold(os.Getenv("NETLIFY"), "true"))
}

// FromNetlifyEnv applies serverless-safe empty defaults explicitly rather
// than relying on the platform to inject a NETLIFY marker at runtime.
func FromNetlifyEnv() Config {
	return fromEnv(true)
}

func fromEnv(netlify bool) Config {
	publicURL, databaseURL := "http://localhost:8080", "postgres://wecratfs:wecratfs@localhost:5432/wecratfs?sslmode=disable"
	redisAddress, meiliAddress, s3Address := "localhost:6379", "localhost:7700", "localhost:9000"
	secureCookies, s3Secure := false, false
	s3AccessKey, s3SecretKey, s3Bucket := "wecratfs", "wecratfs-local-only", "wecratfs-media"
	if netlify {
		publicURL, databaseURL = "", ""
		redisAddress, meiliAddress, s3Address = "", "", ""
		secureCookies, s3Secure = true, true
		s3AccessKey, s3SecretKey, s3Bucket = "", "", ""
	}
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		redisAddress = redisURL
	}
	if s3Endpoint := os.Getenv("S3_ENDPOINT"); s3Endpoint != "" {
		s3Address = s3Endpoint
	}
	return Config{
		Address:            env("WECRATFS_ADDR", ":8080"),
		PublicURL:          env("WECRATFS_PUBLIC_URL", publicURL),
		DatabaseURL:        env("DATABASE_URL", databaseURL),
		DatabaseMaxConns:   int32(intEnv("DATABASE_MAX_CONNS", 2)),
		RedisAddress:       env("REDIS_ADDR", redisAddress),
		MeiliAddress:       env("MEILI_ADDR", meiliAddress),
		MeiliAPIKey:        env("MEILI_API_KEY", ""),
		S3Address:          env("S3_ADDR", s3Address),
		SecureCookies:      boolEnv("WECRATFS_SECURE_COOKIES", secureCookies),
		SMTPAddress:        env("WECRATFS_SMTP_ADDR", ""),
		SMTPUsername:       env("WECRATFS_SMTP_USERNAME", ""),
		SMTPPassword:       env("WECRATFS_SMTP_PASSWORD", ""),
		SMTPFrom:           env("WECRATFS_SMTP_FROM", ""),
		S3AccessKey:        env("S3_ACCESS_KEY", s3AccessKey),
		S3SecretKey:        env("S3_SECRET_KEY", s3SecretKey),
		S3Bucket:           env("S3_BUCKET", s3Bucket),
		S3Secure:           boolEnv("S3_SECURE", s3Secure),
		S3AutoCreateBucket: boolEnv("S3_AUTO_CREATE_BUCKET", !netlify),
	}
}

// ValidateNetlify rejects local defaults and insecure cookie/database settings
// before a function can serve real customer data.
func (c Config) ValidateNetlify() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	database, err := pgxpool.ParseConfig(c.DatabaseURL)
	if err != nil {
		return fmt.Errorf("DATABASE_URL is invalid")
	}
	if isLocalHost(database.ConnConfig.Host) {
		return fmt.Errorf("DATABASE_URL must point to a remote PostgreSQL service")
	}
	databaseURL, err := url.Parse(c.DatabaseURL)
	if err != nil {
		return fmt.Errorf("DATABASE_URL is invalid")
	}
	switch strings.ToLower(databaseURL.Query().Get("sslmode")) {
	case "require", "verify-ca", "verify-full":
	default:
		return fmt.Errorf("DATABASE_URL must explicitly require TLS with sslmode=require or stricter")
	}
	if c.DatabaseMaxConns < 1 || c.DatabaseMaxConns > 20 {
		return fmt.Errorf("DATABASE_MAX_CONNS must be between 1 and 20")
	}
	if strings.TrimSpace(c.RedisAddress) == "" || !strings.HasPrefix(strings.ToLower(c.RedisAddress), "rediss://") || isLocalHost(redisHost(c.RedisAddress)) {
		return fmt.Errorf("REDIS_URL must be a remote TLS Redis URL using rediss://")
	}
	if strings.TrimSpace(c.PublicURL) == "" || !strings.HasPrefix(strings.ToLower(c.PublicURL), "https://") || !c.SecureCookies {
		return fmt.Errorf("WECRATFS_PUBLIC_URL must use HTTPS and secure cookies must be enabled")
	}
	if strings.TrimSpace(c.S3Address) == "" || isLocalHost(s3Host(c.S3Address)) || strings.HasPrefix(strings.ToLower(c.S3Address), "http://") || c.S3AccessKey == "" || c.S3SecretKey == "" || c.S3Bucket == "" || !c.S3Secure {
		return fmt.Errorf("a remote TLS-enabled S3-compatible endpoint and credentials are required")
	}
	if c.S3AutoCreateBucket {
		return fmt.Errorf("S3_AUTO_CREATE_BUCKET must remain disabled in Netlify")
	}
	return nil
}

func redisHost(address string) string {
	if parsed, err := url.Parse(address); err == nil && parsed.Host != "" {
		return parsed.Hostname()
	}
	return address
}

func s3Host(address string) string {
	if parsed, err := url.Parse(address); err == nil && parsed.Host != "" {
		return parsed.Hostname()
	}
	return address
}

func isLocalHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "0.0.0.0" || host == ""
}

func boolEnv(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func intEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
