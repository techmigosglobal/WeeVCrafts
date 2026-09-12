package config

import (
	"os"
	"strconv"
)

type Config struct {
	Address               string
	PublicURL             string
	DatabaseURL           string
	RedisAddress          string
	MeiliAddress          string
	MeiliAPIKey           string
	S3Address             string
	SecureCookies         bool
	RazorpayKeyID         string
	RazorpayKeySecret     string
	RazorpayWebhookSecret string
	RazorpayBaseURL       string
	SMTPAddress           string
	SMTPUsername          string
	SMTPPassword          string
	SMTPFrom              string
	S3AccessKey           string
	S3SecretKey           string
	S3Bucket              string
	S3Secure              bool
}

func FromEnv() Config {
	return Config{
		Address:               env("WECRATFS_ADDR", ":8080"),
		PublicURL:             env("WECRATFS_PUBLIC_URL", "http://localhost:8080"),
		DatabaseURL:           env("DATABASE_URL", "postgres://wecratfs:wecratfs@localhost:5432/wecratfs?sslmode=disable"),
		RedisAddress:          env("REDIS_ADDR", "localhost:6379"),
		MeiliAddress:          env("MEILI_ADDR", "localhost:7700"),
		MeiliAPIKey:           env("MEILI_API_KEY", ""),
		S3Address:             env("S3_ADDR", "localhost:9000"),
		SecureCookies:         boolEnv("WECRATFS_SECURE_COOKIES", false),
		RazorpayKeyID:         env("RAZORPAY_KEY_ID", ""),
		RazorpayKeySecret:     env("RAZORPAY_KEY_SECRET", ""),
		RazorpayWebhookSecret: env("RAZORPAY_WEBHOOK_SECRET", ""),
		RazorpayBaseURL:       env("RAZORPAY_BASE_URL", "https://api.razorpay.com"),
		SMTPAddress:           env("WECRATFS_SMTP_ADDR", ""),
		SMTPUsername:          env("WECRATFS_SMTP_USERNAME", ""),
		SMTPPassword:          env("WECRATFS_SMTP_PASSWORD", ""),
		SMTPFrom:              env("WECRATFS_SMTP_FROM", ""),
		S3AccessKey:           env("S3_ACCESS_KEY", "wecratfs"),
		S3SecretKey:           env("S3_SECRET_KEY", "wecratfs-local-only"),
		S3Bucket:              env("S3_BUCKET", "wecratfs-media"),
		S3Secure:              boolEnv("S3_SECURE", false),
	}
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

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
