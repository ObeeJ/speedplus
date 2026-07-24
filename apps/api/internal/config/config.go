package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port string

	DatabaseURL string

	RedisURL string

	JWTSecret          string
	JWTAccessTTLMin    int
	JWTRefreshTTLDays  int

	SentryDSN string

	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	R2PublicURL       string

	PremblyAPIKey  string
	PremblyBaseURL string
	PremblyAppID   string // dashboard-issued app-id header, NOT a free-text label

	PaystackSecretKey    string
	PaystackWebhookHash  string
	FlutterwaveSecretKey string
	FlutterwaveHash      string
	MonnifyAPIKey        string
	MonnifySecretKey     string
	MonnifyContractCode  string

	BridgeAPIKey    string
	BridgeEnabled   bool

	SendbyteAPIKey     string
	SendbyteFromAddr   string // e.g. "SpeedPlus <noreply@speedplus.app>"

	PINThresholdKobo int64 // require PIN above this amount

	Environment string // "development" | "production"
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:               getEnv("PORT", "8000"),
		DatabaseURL:        mustEnv("DATABASE_URL"),
		RedisURL:           mustEnv("REDIS_URL"),
		JWTSecret:          mustEnv("JWT_SECRET"),
		JWTAccessTTLMin:    getEnvInt("JWT_ACCESS_TTL_MIN", 15),
		JWTRefreshTTLDays:  getEnvInt("JWT_REFRESH_TTL_DAYS", 30),
		SentryDSN:          getEnv("SENTRY_DSN", ""),
		R2AccountID:        getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:      getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretAccessKey:  getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2BucketName:       getEnv("R2_BUCKET_NAME", "speedplus-docs"),
		R2PublicURL:        getEnv("R2_PUBLIC_URL", ""),
		PremblyAPIKey:      getEnv("PREMBLY_API_KEY", ""),
		PremblyBaseURL:     getEnv("PREMBLY_BASE_URL", "https://api.prembly.com"),
		PremblyAppID:       getEnv("PREMBLY_APP_ID", ""),
		PaystackSecretKey:  getEnv("PAYSTACK_SECRET_KEY", ""),
		PaystackWebhookHash: getEnv("PAYSTACK_WEBHOOK_HASH", ""),
		FlutterwaveSecretKey: getEnv("FLUTTERWAVE_SECRET_KEY", ""),
		FlutterwaveHash:    getEnv("FLUTTERWAVE_HASH", ""),
		MonnifyAPIKey:      getEnv("MONNIFY_API_KEY", ""),
		MonnifySecretKey:   getEnv("MONNIFY_SECRET_KEY", ""),
		MonnifyContractCode: getEnv("MONNIFY_CONTRACT_CODE", ""),
		BridgeAPIKey:       getEnv("BRIDGE_API_KEY", ""),
		BridgeEnabled:      getEnv("BRIDGE_ENABLED", "false") == "true",
		SendbyteAPIKey:     getEnv("SENDBYTE_API_KEY", ""),
		SendbyteFromAddr:   getEnv("SENDBYTE_FROM_ADDRESS", "SpeedPlus <noreply@speedplus.app>"),
		PINThresholdKobo:   int64(getEnvInt("PIN_THRESHOLD_KOBO", 5000000)), // ₦50,000
		Environment:        getEnv("ENVIRONMENT", "development"),
	}

	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 bytes")
	}

	// In production every payment provider key must be set.
	// A missing key means every charge silently fails — fail fast at boot.
	if cfg.Environment == "production" {
		required := map[string]string{
			"PAYSTACK_SECRET_KEY":   cfg.PaystackSecretKey,
			"MONNIFY_API_KEY":       cfg.MonnifyAPIKey,
			"MONNIFY_SECRET_KEY":    cfg.MonnifySecretKey,
			"MONNIFY_CONTRACT_CODE": cfg.MonnifyContractCode,
		}
		for k, v := range required {
			if v == "" {
				return nil, fmt.Errorf("required env var %s is not set in production", k)
			}
		}
	}

	return cfg, nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required env var %s is not set", key))
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
