package platform

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"strings"
)

type Config struct {
	Production                                                                    bool
	Origin, Addr, MongoURI, MongoDatabase, DataDir, EpheDir, FontPath, ChromePath string
	PaymentMode, KeyID, KeySecret, WebhookSecret                                  string
	ZoomMode, ZoomAccount, ZoomClient, ZoomSecret, ZoomUser                       string
	OTPMode, OTPURL, OTPToken, NotificationURL, NotificationToken                 string
	AdminPassword, AdminTOTP                                                      string
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
	c := Config{Production: os.Getenv("APP_ENV") == "production", Origin: env("APP_ORIGIN", "http://localhost:5173"), Addr: env("HTTP_ADDR", "127.0.0.1:8080"), MongoURI: os.Getenv("MONGODB_URI"), MongoDatabase: env("MONGODB_DATABASE", "mishra4you_development"), DataDir: env("DATA_DIR", ".local"), EpheDir: env("EPHE_DIR", "ephe"), FontPath: env("FONT_PATH", "assets/NotoSansDevanagari-Regular.ttf"), ChromePath: os.Getenv("CHROME_PATH"), PaymentMode: env("PAYMENT_MODE", "simulation"), KeyID: os.Getenv("RAZORPAY_KEY_ID"), KeySecret: os.Getenv("RAZORPAY_KEY_SECRET"), WebhookSecret: os.Getenv("RAZORPAY_WEBHOOK_SECRET"), ZoomMode: env("ZOOM_MODE", "simulation"), ZoomAccount: os.Getenv("ZOOM_ACCOUNT_ID"), ZoomClient: os.Getenv("ZOOM_CLIENT_ID"), ZoomSecret: os.Getenv("ZOOM_CLIENT_SECRET"), ZoomUser: env("ZOOM_USER_ID", "me"), OTPMode: env("OTP_MODE", "simulation"), OTPURL: os.Getenv("OTP_GATEWAY_URL"), OTPToken: os.Getenv("OTP_GATEWAY_TOKEN"), NotificationURL: os.Getenv("NOTIFICATION_URL"), NotificationToken: os.Getenv("NOTIFICATION_TOKEN"), AdminPassword: env("ADMIN_PASSWORD", "local-admin-change-me"), AdminTOTP: os.Getenv("ADMIN_TOTP_SECRET")}
	if c.PaymentMode != "simulation" && c.PaymentMode != "razorpay" {
		return c, fmt.Errorf("invalid PAYMENT_MODE")
	}
	if c.ZoomMode != "simulation" && c.ZoomMode != "zoom" {
		return c, fmt.Errorf("invalid ZOOM_MODE")
	}
	if c.OTPMode != "simulation" && c.OTPMode != "http" {
		return c, fmt.Errorf("invalid OTP_MODE")
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
	if c.OTPMode == "http" && (!strings.HasPrefix(c.OTPURL, "https://") || c.OTPToken == "") {
		return c, fmt.Errorf("HTTPS OTP gateway and token required")
	}
	if c.Production {
		if c.MongoURI == "" || !strings.HasPrefix(c.Origin, "https://") || len(c.AdminPassword) < 20 || c.AdminPassword == "local-admin-change-me" || c.AdminTOTP == "" {
			return c, fmt.Errorf("production requires MongoDB Atlas, HTTPS origin, strong admin password and TOTP")
		}
		if c.OTPMode == "simulation" || c.PaymentMode == "simulation" || c.ZoomMode == "simulation" {
			return c, fmt.Errorf("simulations are forbidden in production")
		}
		if !strings.HasPrefix(c.NotificationURL, "https://") || c.NotificationToken == "" {
			return c, fmt.Errorf("production notification gateway required")
		}
	}
	return c, nil
}

// env reads an environment setting, using its supplied default when no value is configured.
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
