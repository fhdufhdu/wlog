package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BindAddr, DatabaseURL, PublicBaseURL, SiteName, SiteDescription, DefaultSocialImage string
	AdminUsername, AdminPasswordHash, SessionSecret, UploadDir                          string
	SecureCookie                                                                        bool
	MigrationBaselineExisting                                                           bool
	MaxUploadBytes                                                                      int64
	ImageMaxDimension                                                                   int
	ImageMaxPixels                                                                      int64
	ImageWebPQuality                                                                    float32
	ImageOrphanGraceHours                                                               int
}

func FromEnv() (Config, error) {
	baseURL, err := required("PUBLIC_BASE_URL")
	if err != nil {
		return Config{}, err
	}
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return Config{}, fmt.Errorf("PUBLIC_BASE_URL이 올바르지 않습니다")
	}
	databaseURL, err := required("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}
	username, err := required("ADMIN_USERNAME")
	if err != nil {
		return Config{}, err
	}
	passwordHash, err := required("ADMIN_PASSWORD_HASH")
	if err != nil {
		return Config{}, err
	}
	secret, err := required("SESSION_SECRET")
	if err != nil {
		return Config{}, err
	}
	if len(secret) < 32 {
		return Config{}, fmt.Errorf("SESSION_SECRET은 32바이트 이상이어야 합니다")
	}
	maxUpload, err := intValue("MAX_UPLOAD_BYTES", 5_242_880)
	if err != nil {
		return Config{}, err
	}
	maxDimension, err := intValue("IMAGE_MAX_DIMENSION", 2560)
	if err != nil || maxDimension < 320 || maxDimension > 8192 {
		return Config{}, fmt.Errorf("IMAGE_MAX_DIMENSION은 320 이상 8192 이하여야 합니다")
	}
	maxPixels, err := intValue("IMAGE_MAX_PIXELS", 40_000_000)
	if err != nil || maxPixels < 1_000_000 || maxPixels > 100_000_000 {
		return Config{}, fmt.Errorf("IMAGE_MAX_PIXELS는 1000000 이상 100000000 이하여야 합니다")
	}
	quality, err := floatValue("IMAGE_WEBP_QUALITY", 82)
	if err != nil || quality < 1 || quality > 100 {
		return Config{}, fmt.Errorf("IMAGE_WEBP_QUALITY는 1 이상 100 이하여야 합니다")
	}
	grace, err := intValue("IMAGE_ORPHAN_GRACE_HOURS", 24)
	if err != nil || grace < 1 {
		return Config{}, fmt.Errorf("IMAGE_ORPHAN_GRACE_HOURS는 1 이상이어야 합니다")
	}
	secure, err := boolValue("SECURE_COOKIE", parsed.Scheme == "https")
	if err != nil {
		return Config{}, err
	}
	baselineExisting, err := boolValue("MIGRATION_BASELINE_EXISTING", false)
	if err != nil {
		return Config{}, fmt.Errorf("MIGRATION_BASELINE_EXISTING: %w", err)
	}
	return Config{
		BindAddr: value("BIND_ADDR", "127.0.0.1:3000"), DatabaseURL: databaseURL,
		PublicBaseURL: strings.TrimRight(baseURL, "/"), SiteName: value("SITE_NAME", "Wlog"), SiteDescription: value("SITE_DESCRIPTION", "개발하고 운영하며 알게 된 것을 기록합니다."), DefaultSocialImage: os.Getenv("DEFAULT_SOCIAL_IMAGE"),
		AdminUsername: username, AdminPasswordHash: passwordHash, SessionSecret: secret, UploadDir: value("UPLOAD_DIR", "uploads"), SecureCookie: secure, MigrationBaselineExisting: baselineExisting,
		MaxUploadBytes: int64(maxUpload), ImageMaxDimension: maxDimension, ImageMaxPixels: int64(maxPixels), ImageWebPQuality: float32(quality), ImageOrphanGraceHours: grace,
	}, nil
}

func required(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("%s이 필요합니다", key)
	}
	return v, nil
}
func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func intValue(key string, fallback int) (int, error) {
	return strconv.Atoi(value(key, strconv.Itoa(fallback)))
}
func floatValue(key string, fallback float64) (float64, error) {
	return strconv.ParseFloat(value(key, strconv.FormatFloat(fallback, 'f', -1, 64)), 32)
}
func boolValue(key string, fallback bool) (bool, error) {
	return strconv.ParseBool(value(key, strconv.FormatBool(fallback)))
}
