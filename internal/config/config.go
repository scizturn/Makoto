package config

import (
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
)

type Config struct {
	Mode               string
	Timezone           string
	RateLimitPerMinute int
	DatabaseDSN        string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	QueueName          string
	DeadLetterQueue    string
	MaxAttempts        int
	KirimEmailUsername string
	KirimEmailAPIToken string
	KirimEmailBaseURL  string
	KirimEmailDomain   string
	KirimEmailValidate bool
	FromEmail          string
	FromName           string
	TemplateIDs        []string
	EmailTemplateDir   string
	EmailSubject       string
	ActionURL          string
	KyouIDAPIBaseURL   string
	KyouIDAPIToken     string
	DiscordWebhookURL  string
	DiscordEnabled     bool
	AnniversaryEnabled           bool
	AnniversaryQueueName         string
	AnniversaryDeadLetterQueue   string
	AnniversaryTemplateIDs       []string
	AnniversaryEmailTemplateDir  string
	AnniversaryEmailSubject      string
	AnniversaryEmailSubjects     []string
	LeftoverCartEnabled          bool
	LeftoverCartQueueName        string
	LeftoverCartDeadLetterQueue  string
	LeftoverCartTemplateIDs      []string
	LeftoverCartEmailTemplateDir string
	LeftoverCartEmailSubject     string
	LeftoverCartGreetings        []string
	LeftoverCartURL              string
	DiscountedWishlistEnabled          bool
	DiscountedWishlistQueueName        string
	DiscountedWishlistDeadLetterQueue  string
	DiscountedWishlistTemplateIDs      []string
	DiscountedWishlistEmailTemplateDir string
	DiscountedWishlistEmailSubject     string
	DiscountedWishlistGreetings        []string
	DiscountedWishlistURL              string
	WinbackEnabled          bool
	WinbackQueueName        string
	WinbackDeadLetterQueue  string
	WinbackTemplateIDs      []string
	WinbackEmailTemplateDir string
	WinbackEmailSubject     string
	WinbackEmailSubjects    []string
	WinbackGreetings        []string
	WinbackActionURL        string
}

func Load() Config {
	return Config{
		Mode:               env("MAKOTO_MODE", "run-once"),
		Timezone:           env("MAKOTO_TIMEZONE", "Asia/Jakarta"),
		RateLimitPerMinute: envInt("MAKOTO_RATE_LIMIT_PER_MINUTE", 100),
		DatabaseDSN:        oldDatabaseDSN(),
		RedisAddr:          env("REDIS_ADDR", "redis:6379"),
		RedisPassword:      os.Getenv("REDIS_PASSWORD"),
		RedisDB:            envIntAllowZero("REDIS_DB", 0),
		QueueName:          env("MAKOTO_QUEUE_NAME", "birthday_email_jobs"),
		DeadLetterQueue:    env("MAKOTO_DEAD_LETTER_QUEUE", "birthday_email_jobs_dead"),
		MaxAttempts:        envInt("MAKOTO_MAX_ATTEMPTS", 3),
		KirimEmailUsername: os.Getenv("KIRIM_EMAIL_USERNAME"),
		KirimEmailAPIToken: os.Getenv("KIRIM_EMAIL_API_TOKEN"),
		KirimEmailBaseURL:  env("KIRIM_EMAIL_BASE_URL", "https://smtp-app.kirim.email"),
		KirimEmailDomain:   env("KIRIM_EMAIL_DOMAIN", "kyou.id"),
		KirimEmailValidate: envBool("KIRIM_EMAIL_VALIDATE", false),
		FromEmail:          env("KIRIM_EMAIL_FROM_EMAIL", "nandayo@kyou.id"),
		FromName:           env("KIRIM_EMAIL_FROM_NAME", "Kyou.id"),
		TemplateIDs:        envList("MAKOTO_TEMPLATE_IDS", []string{"tpl_001", "tpl_002", "tpl_003"}),
		EmailTemplateDir:   os.Getenv("MAKOTO_EMAIL_TEMPLATE_DIR"),
		EmailSubject:       env("MAKOTO_EMAIL_SUBJECT", "Selamat ulang tahun, {{ .Name }}"),
		ActionURL:          env("MAKOTO_ACTION_URL", "https://kyou.id/user/my-voucher"),
		KyouIDAPIBaseURL:   env("KYOU_ID_API_BASE_URL", "https://kyou.id"),
		KyouIDAPIToken:     os.Getenv("KYOU_ID_API_TOKEN"),
		DiscordWebhookURL:  os.Getenv("DISCORD_WEBHOOK_URL"),
		DiscordEnabled:     envBool("DISCORD_WEBHOOK_ENABLED", true),
		AnniversaryEnabled:          envBool("MAKOTO_ANNIVERSARY_ENABLED", false),
		AnniversaryQueueName:        env("MAKOTO_ANNIVERSARY_QUEUE_NAME", "anniversary_email_jobs"),
		AnniversaryDeadLetterQueue:  env("MAKOTO_ANNIVERSARY_DEAD_LETTER_QUEUE", "anniversary_email_jobs_dead"),
		AnniversaryTemplateIDs:      envList("MAKOTO_ANNIVERSARY_TEMPLATE_IDS", []string{"anniversary1.html", "anniversary2.html", "anniversary3.html"}),
		AnniversaryEmailTemplateDir: os.Getenv("MAKOTO_ANNIVERSARY_EMAIL_TEMPLATE_DIR"),
		AnniversaryEmailSubject:     env("MAKOTO_ANNIVERSARY_EMAIL_SUBJECT", "Happy Anniversary, {{ .Name }}! 🎉"),
		AnniversaryEmailSubjects: envListPipe("MAKOTO_ANNIVERSARY_EMAIL_SUBJECTS", []string{
			"Cieee anniversary! Cek hadiah dari Kyou buat nambahin khilafanmu!",
			"Ada kado spesial buat anniversary ke-{{ .Years }}, {{ .FirstName }}!",
			"Kejutan spesial untuk anniversary kamu, {{ .FirstName }}.",
		}),
		LeftoverCartEnabled:          envBool("MAKOTO_LEFTOVER_CART_ENABLED", false),
		LeftoverCartQueueName:        env("MAKOTO_LEFTOVER_CART_QUEUE_NAME", "leftover_cart_email_jobs"),
		LeftoverCartDeadLetterQueue:  env("MAKOTO_LEFTOVER_CART_DEAD_LETTER_QUEUE", "leftover_cart_email_jobs_dead"),
		LeftoverCartTemplateIDs:      envList("MAKOTO_LEFTOVER_CART_TEMPLATE_IDS", []string{"leftover_cart1.html", "leftover_cart2.html", "leftover_cart3.html", "leftover_cart4.html", "leftover_cart5.html"}),
		LeftoverCartEmailTemplateDir: os.Getenv("MAKOTO_LEFTOVER_CART_EMAIL_TEMPLATE_DIR"),
		LeftoverCartEmailSubject:     env("MAKOTO_LEFTOVER_CART_EMAIL_SUBJECT", "Eh {{ .FirstName }}, keranjangmu masih nunggu nih!"),
		LeftoverCartGreetings: envListPipe("MAKOTO_LEFTOVER_CART_GREETINGS", []string{
			"Hei {{ .FirstName }}, kamu lagi sibuk ya?",
			"Woi {{ .FirstName }}, jangan lupa sama keranjangmu!",
			"{{ .FirstName }}, barang incaran kamu masih ada nih!",
			"Psst {{ .FirstName }}, keranjangmu masih nunggu kamu balik!",
			"{{ .FirstName }}, sayang banget kalau sampai kehabisan!",
		}),
		LeftoverCartURL: env("MAKOTO_LEFTOVER_CART_URL", "https://kyou.id/user/cart"),
		DiscountedWishlistEnabled:          envBool("MAKOTO_DISCOUNTED_WISHLIST_ENABLED", false),
		DiscountedWishlistQueueName:        env("MAKOTO_DISCOUNTED_WISHLIST_QUEUE_NAME", "discounted_wishlist_email_jobs"),
		DiscountedWishlistDeadLetterQueue:  env("MAKOTO_DISCOUNTED_WISHLIST_DEAD_LETTER_QUEUE", "discounted_wishlist_email_jobs_dead"),
		DiscountedWishlistTemplateIDs:      envList("MAKOTO_DISCOUNTED_WISHLIST_TEMPLATE_IDS", []string{"discounted_wishlist1.html", "discounted_wishlist2.html", "discounted_wishlist3.html"}),
		DiscountedWishlistEmailTemplateDir: os.Getenv("MAKOTO_DISCOUNTED_WISHLIST_EMAIL_TEMPLATE_DIR"),
		DiscountedWishlistEmailSubject:     env("MAKOTO_DISCOUNTED_WISHLIST_EMAIL_SUBJECT", "{{ .FirstName }}, wishlist kamu lagi diskon nih!"),
		DiscountedWishlistGreetings: envListPipe("MAKOTO_DISCOUNTED_WISHLIST_GREETINGS", []string{
			"Psst {{ .FirstName }}, wishlist incaran kamu lagi diskon nih!",
			"Hei {{ .FirstName }}, ada kabar bagus buat koleksimu!",
			"{{ .FirstName }}, saatnya checkout wishlist yang udah lama kamu incer!",
			"Woi {{ .FirstName }}, item wishlist favoritmu lagi ada promo!",
			"{{ .FirstName }}, jangan sampai nyesel — wishlistmu lagi diskon sekarang!",
		}),
		DiscountedWishlistURL: env("MAKOTO_DISCOUNTED_WISHLIST_URL", "https://kyou.id/user/wishlist"),
		WinbackEnabled:          envBool("MAKOTO_WINBACK_ENABLED", false),
		WinbackQueueName:        env("MAKOTO_WINBACK_QUEUE_NAME", "winback_email_jobs"),
		WinbackDeadLetterQueue:  env("MAKOTO_WINBACK_DEAD_LETTER_QUEUE", "winback_email_jobs_dead"),
		WinbackTemplateIDs:      envList("MAKOTO_WINBACK_TEMPLATE_IDS", []string{"winback1.html", "winback2.html", "winback3.html"}),
		WinbackEmailTemplateDir: os.Getenv("MAKOTO_WINBACK_EMAIL_TEMPLATE_DIR"),
		WinbackEmailSubject: env("MAKOTO_WINBACK_EMAIL_SUBJECT", "{{ .FirstName }}, kita kangen kamu nih!"),
		WinbackEmailSubjects: envListPipe("MAKOTO_WINBACK_EMAIL_SUBJECTS", []string{
			"{{ .FirstName }}, kita kangen kamu nih!",
			"Ada voucher spesial buat kamu, {{ .FirstName }}!",
			"{{ .FirstName }}, udah lama nggak belanja di Kyou nih...",
		}),
		WinbackGreetings: envListPipe("MAKOTO_WINBACK_GREETINGS", []string{
			"Hei {{ .FirstName }}, udah lama banget nih nggak ketemu!",
			"{{ .FirstName }}, kita kangen sama kamu!",
			"Lama nggak belanja di Kyou, {{ .FirstName }}?",
			"{{ .FirstName }}, ada yang nungguin kamu balik nih!",
			"Psst {{ .FirstName }}, ada voucher spesial buat kamu!",
		}),
		WinbackActionURL: env("MAKOTO_WINBACK_ACTION_URL", "https://kyou.id/user/my-voucher"),
	}
}

func oldDatabaseDSN() string {
	host := env("OLD_DATABASE_HOST", "")
	name := env("OLD_DATABASE_NAME", "")
	username := env("OLD_DATABASE_USERNAME", "")
	password := os.Getenv("OLD_DATABASE_PASSWORD")
	if host == "" || name == "" || username == "" {
		return ""
	}

	cfg := mysql.Config{
		User:                 username,
		Passwd:               password,
		Net:                  "tcp",
		Addr:                 net.JoinHostPort(host, env("OLD_DATABASE_PORT", "3306")),
		DBName:               name,
		ParseTime:            true,
		AllowNativePasswords: true,
		Params: map[string]string{
			"charset":   "utf8mb4",
			"collation": "utf8mb4_unicode_ci",
		},
	}
	return cfg.FormatDSN()
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

func envListPipe(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parts := strings.Split(value, "|")
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
