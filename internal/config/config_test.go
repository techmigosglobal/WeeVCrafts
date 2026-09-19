package config

import "testing"

func validNetlifyConfig() Config {
	return Config{
		PublicURL:        "https://shop.example.net",
		DatabaseURL:      "postgres://app:secret@pg.example.net:5432/store?sslmode=require",
		DatabaseMaxConns: 2,
		RedisAddress:     "rediss://:secret@redis.example.net:6380/0",
		S3Address:        "https://objects.example.net",
		S3AccessKey:      "key",
		S3SecretKey:      "secret",
		S3Bucket:         "weevcrafts-media",
		S3Secure:         true,
		SecureCookies:    true,
	}
}

func TestValidateNetlifyAcceptsRemoteTLSConfiguration(t *testing.T) {
	if err := validNetlifyConfig().ValidateNetlify(); err != nil {
		t.Fatalf("valid Netlify config rejected: %v", err)
	}
}

func TestValidateNetlifyNeedsOnlyManagedPostgres(t *testing.T) {
	cfg := Config{
		PublicURL:        "https://shop.example.net",
		DatabaseURL:      "postgres://app:secret@pg.example.net:5432/store?sslmode=require",
		DatabaseMaxConns: 1,
		SecureCookies:    true,
	}
	if err := cfg.ValidateNetlify(); err != nil {
		t.Fatalf("managed PostgreSQL config rejected without Redis or S3: %v", err)
	}
}

func TestFromNetlifyEnvUsesManagedDatabaseURL(t *testing.T) {
	t.Setenv("NETLIFY_DB_URL", "postgres://app:secret@pg.example.net:5432/store?sslmode=require")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("NETLIFY", "false")
	t.Setenv("WECRATFS_PUBLIC_URL", "https://shop.example.net")
	cfg := FromNetlifyEnv()
	if cfg.DatabaseURL == "" || cfg.PublicURL != "https://shop.example.net" || !cfg.SecureCookies {
		t.Fatal("Netlify environment did not select the managed database and safe HTTPS defaults")
	}
	if err := cfg.ValidateNetlify(); err != nil {
		t.Fatalf("managed database environment rejected: %v", err)
	}
}

func TestFromEnvNetlifyDoesNotInjectLocalOrDemoDefaults(t *testing.T) {
	for key, value := range map[string]string{
		"NETLIFY": "true", "DATABASE_URL": "", "NETLIFY_DB_URL": "", "DATABASE_MAX_CONNS": "",
		"REDIS_URL": "", "REDIS_ADDR": "", "MEILI_ADDR": "", "MEILI_API_KEY": "",
		"S3_ENDPOINT": "", "S3_ADDR": "", "S3_ACCESS_KEY": "", "S3_SECRET_KEY": "",
		"S3_BUCKET": "", "S3_AUTO_CREATE_BUCKET": "", "WECRATFS_PUBLIC_URL": "",
		"WECRATFS_SECURE_COOKIES": "", "S3_SECURE": "",
	} {
		t.Setenv(key, value)
	}
	cfg := FromEnv()
	if cfg.DatabaseURL != "" || cfg.RedisAddress != "" || cfg.MeiliAddress != "" || cfg.S3Address != "" || cfg.S3Bucket != "" {
		t.Fatalf("Netlify config injected a local service default: %#v", cfg)
	}
	if cfg.SecureCookies != true || !cfg.S3Secure || cfg.S3AutoCreateBucket {
		t.Fatalf("Netlify config has unsafe defaults: %#v", cfg)
	}
}

func TestFromNetlifyEnvDoesNotDependOnPlatformMarker(t *testing.T) {
	t.Setenv("NETLIFY", "false")
	for _, key := range []string{
		"DATABASE_URL", "NETLIFY_DB_URL", "REDIS_URL", "REDIS_ADDR", "MEILI_ADDR", "S3_ENDPOINT", "S3_ADDR",
		"S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET", "S3_AUTO_CREATE_BUCKET",
	} {
		t.Setenv(key, "")
	}
	cfg := FromNetlifyEnv()
	if cfg.DatabaseURL != "" || cfg.RedisAddress != "" || cfg.S3Address != "" || cfg.S3Bucket != "" || cfg.S3AutoCreateBucket {
		t.Fatalf("Netlify-safe configuration depended on NETLIFY marker: %#v", cfg)
	}
}

func TestValidateNetlifyRejectsLocalOrInsecureDependencies(t *testing.T) {
	tests := map[string]func(*Config){
		"missing database":              func(c *Config) { c.DatabaseURL = "" },
		"local database":                func(c *Config) { c.DatabaseURL = "postgres://app:secret@localhost:5432/store?sslmode=require" },
		"database without explicit TLS": func(c *Config) { c.DatabaseURL = "postgres://app:secret@pg.example.net:5432/store" },
		"insecure public URL":           func(c *Config) { c.PublicURL = "http://shop.example.net" },
		"insecure cookie":               func(c *Config) { c.SecureCookies = false },
		"unbounded connection pool":     func(c *Config) { c.DatabaseMaxConns = 21 },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := validNetlifyConfig()
			mutate(&cfg)
			if err := cfg.ValidateNetlify(); err == nil {
				t.Fatal("expected invalid configuration to be rejected")
			}
		})
	}
}
