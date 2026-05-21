package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Mode               string
	Timezone           string
	RateLimitPerMinute int
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	QueueName          string
	KirimEmailUsername string
	KirimEmailAPIToken string
	KirimEmailBaseURL  string
	KirimEmailDomain   string
	FromEmail          string
	FromName           string
	TemplateIDs        []string
	KyouIDAPIBaseURL   string
	KyouIDAPIToken     string
	DiscordWebhookURL  string
	DiscordEnabled     bool
}

func Load() Config {
	return Config{
		Mode:               env("MAKOTO_MODE", "run-once"),
		Timezone:           env("MAKOTO_TIMEZONE", "Asia/Jakarta"),
		RateLimitPerMinute: envInt("MAKOTO_RATE_LIMIT_PER_MINUTE", 100),
		RedisAddr:          env("REDIS_ADDR", "redis:6379"),
		RedisPassword:      os.Getenv("REDIS_PASSWORD"),
		RedisDB:            envIntAllowZero("REDIS_DB", 0),
		QueueName:          env("MAKOTO_QUEUE_NAME", "birthday_email_jobs"),
		KirimEmailUsername: os.Getenv("KIRIM_EMAIL_USERNAME"),
		KirimEmailAPIToken: os.Getenv("KIRIM_EMAIL_API_TOKEN"),
		KirimEmailBaseURL:  env("KIRIM_EMAIL_BASE_URL", "https://smtp-app.kirim.email"),
		KirimEmailDomain:   env("KIRIM_EMAIL_DOMAIN", "kyou.id"),
		FromEmail:          env("KIRIM_EMAIL_FROM_EMAIL", "nandayo@kyou.id"),
		FromName:           env("KIRIM_EMAIL_FROM_NAME", "Kyou.id"),
		TemplateIDs:        envList("MAKOTO_TEMPLATE_IDS", []string{"tpl_001", "tpl_002", "tpl_003"}),
		KyouIDAPIBaseURL:   env("KYOU_ID_API_BASE_URL", "https://kyou.id"),
		KyouIDAPIToken:     os.Getenv("KYOU_ID_API_TOKEN"),
		DiscordWebhookURL:  os.Getenv("DISCORD_WEBHOOK_URL"),
		DiscordEnabled:     envBool("DISCORD_WEBHOOK_ENABLED", true),
	}
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func envIntAllowZero(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func envList(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return fallback
	}
	return items
}
