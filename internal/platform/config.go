package platform

import (
	"fmt"
	"github.com/joho/godotenv"
	"net/mail"
	"os"
	"strings"
)

type Config struct {
	Production                                                                    bool
	TrustLoopbackProxy                                                            bool
	Origin, Addr, MongoURI, MongoDatabase, DataDir, EpheDir, FontPath, ChromePath string
	PaymentMode, KeyID, KeySecret, WebhookSecret                                  string
	ZoomMode, ZoomAccount, ZoomClient, ZoomSecret, ZoomUser                       string
	NotificationURL, NotificationToken                                            string
	AdminEmail, AdminPassword                                                     string
}

// LoadConfig loads the selected environment file and validates storage, provider and production settings.
func LoadConfig() (Config, error) {
	if file := env("ENV_FILE", ".env"); file != "none" {
		if err := godotenv.Load(file); err != nil && !os.IsNotExist(err) {
			return Config{}, fmt.Errorf("could not parse environment file (contents redacted)")
		}
	}
	mode := env("APP_ENV", "development")
	if mode != "development" && mode != "test" && mode != "production" {
		return Config{}, fmt.Errorf("APP_ENV must be development, test or production")
	}
	c := Config{Production: os.Getenv("APP_ENV") == "production", Origin: env("APP_ORIGIN", "http://localhost:5173"), Addr: env("HTTP_ADDR", "127.0.0.1:8080"), MongoURI: os.Getenv("MONGODB_URI"), MongoDatabase: env("MONGODB_DATABASE", "mishra4you_development"), DataDir: env("DATA_DIR", ".local"), EpheDir: env("EPHE_DIR", "ephe"), FontPath: env("FONT_PATH", "assets/NotoSansDevanagari-Regular.ttf"), ChromePath: os.Getenv("CHROME_PATH"), PaymentMode: env("PAYMENT_MODE", "simulation"), KeyID: os.Getenv("RAZORPAY_KEY_ID"), KeySecret: os.Getenv("RAZORPAY_KEY_SECRET"), WebhookSecret: os.Getenv("RAZORPAY_WEBHOOK_SECRET"), ZoomMode: env("ZOOM_MODE", "simulation"), ZoomAccount: os.Getenv("ZOOM_ACCOUNT_ID"), ZoomClient: os.Getenv("ZOOM_CLIENT_ID"), ZoomSecret: os.Getenv("ZOOM_CLIENT_SECRET"), ZoomUser: env("ZOOM_USER_ID", "me"), NotificationURL: os.Getenv("NOTIFICATION_URL"), NotificationToken: os.Getenv("NOTIFICATION_TOKEN"), AdminEmail: strings.ToLower(strings.TrimSpace(env("ADMIN_EMAIL", "local-admin@example.test"))), AdminPassword: env("ADMIN_PASSWORD", "local-admin-change-me")}
	proxyTrust := env("TRUST_LOOPBACK_PROXY", "false")
	if proxyTrust != "true" && proxyTrust != "false" {
		return c, fmt.Errorf("TRUST_LOOPBACK_PROXY must be true or false")
	}
	c.TrustLoopbackProxy = proxyTrust == "true"
	if c.PaymentMode != "simulation" && c.PaymentMode != "razorpay" {
		return c, fmt.Errorf("invalid PAYMENT_MODE")
	}
	if c.ZoomMode != "simulation" && c.ZoomMode != "zoom" {
		return c, fmt.Errorf("invalid ZOOM_MODE")
	}
	if c.PaymentMode == "razorpay" && (c.KeyID == "" || c.KeySecret == "" || c.WebhookSecret == "") {
		return c, fmt.Errorf("Razorpay credentials incomplete")
	}
	if c.PaymentMode == "razorpay" && strings.HasPrefix(c.KeyID, "rzp_live_") && (!c.Production || os.Getenv("ALLOW_LIVE_PAYMENTS") != "true") {
		return c, fmt.Errorf("live Razorpay keys require production and explicit ALLOW_LIVE_PAYMENTS=true")
	}
	if c.ZoomMode == "zoom" && (c.ZoomAccount == "" || c.ZoomClient == "" || c.ZoomSecret == "") {
		return c, fmt.Errorf("Zoom credentials incomplete")
	}
	if c.Production {
		if c.MongoURI == "" || !strings.HasPrefix(c.Origin, "https://") {
			return c, fmt.Errorf("production requires MongoDB Atlas and an HTTPS origin")
		}
		address, err := mail.ParseAddress(c.AdminEmail)
		if err != nil || address.Address != c.AdminEmail || c.AdminEmail == "local-admin@example.test" || len(c.AdminPassword) < 8 || templateCredential(c.AdminPassword) {
			return c, fmt.Errorf("production requires a configured admin email and a password of at least 8 characters")
		}
		if c.PaymentMode == "simulation" || c.ZoomMode == "simulation" {
			return c, fmt.Errorf("simulations are forbidden in production")
		}
		if !strings.HasPrefix(c.NotificationURL, "https://") || c.NotificationToken == "" {
			return c, fmt.Errorf("production notification gateway required")
		}
	}
	return c, nil
}

// templateCredential identifies published example credentials; it never logs their contents.
func templateCredential(value string) bool {
	value = strings.ToUpper(strings.TrimSpace(value))
	return strings.Contains(value, "REPLACE_") || strings.Contains(value, "CHANGE-ME") || strings.Contains(value, "CHANGE_ME")
}

// paymentConfigured prevents checkout with empty or publicly documented example provider secrets.
func paymentConfigured(c Config) bool {
	if c.PaymentMode != "razorpay" {
		return !c.Production
	}
	for _, value := range []string{c.KeyID, c.KeySecret, c.WebhookSecret} {
		if strings.TrimSpace(value) == "" || templateCredential(value) {
			return false
		}
	}
	return true
}

// env reads an environment setting, using its supplied default when no value is configured.
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
