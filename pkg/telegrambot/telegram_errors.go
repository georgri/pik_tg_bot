package telegrambot

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/georgri/pik_tg_bot/pkg/util"
)

type TelegramAPIError struct {
	Method string
	Reason string

	URL         string
	StatusCode  int
	Status      string
	ContentType string

	// Telegram-level fields (usually duplicated into body JSON)
	TelegramErrorCode   int
	TelegramDescription string

	BodySnippet string

	// Debug-only, safe token metadata (never the token itself).
	TokenInfo string

	// Human-oriented hint for fast diagnosis.
	Hint string
}

func (e *TelegramAPIError) Error() string {
	if e == nil {
		return "<nil>"
	}

	parts := make([]string, 0, 10)
	if e.Method != "" {
		parts = append(parts, fmt.Sprintf("telegram %s failed", e.Method))
	} else {
		parts = append(parts, "telegram API call failed")
	}
	if e.Reason != "" {
		parts = append(parts, fmt.Sprintf("reason=%s", e.Reason))
	}
	if e.URL != "" {
		parts = append(parts, fmt.Sprintf("url=%s", e.URL))
	}
	if e.Status != "" {
		parts = append(parts, fmt.Sprintf("http=%s", e.Status))
	} else if e.StatusCode != 0 {
		parts = append(parts, fmt.Sprintf("http_status=%d", e.StatusCode))
	}
	if e.ContentType != "" {
		parts = append(parts, fmt.Sprintf("content-type=%s", e.ContentType))
	}
	if e.TelegramErrorCode != 0 || e.TelegramDescription != "" {
		parts = append(parts, fmt.Sprintf("telegram_error=%d %q", e.TelegramErrorCode, e.TelegramDescription))
	}
	if e.TokenInfo != "" {
		parts = append(parts, fmt.Sprintf("token=%s", e.TokenInfo))
	}
	if e.BodySnippet != "" {
		parts = append(parts, fmt.Sprintf("body=%q", e.BodySnippet))
	}
	if e.Hint != "" {
		parts = append(parts, fmt.Sprintf("hint=%q", e.Hint))
	}

	return strings.Join(parts, "; ")
}

func telegramBodySnippet(body []byte, maxLen int) string {
	if maxLen <= 0 || len(body) == 0 {
		return ""
	}
	s := strings.TrimSpace(string(body))
	if s == "" {
		return ""
	}
	// Reduce log noise from pretty JSON.
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxLen {
		return s[:maxLen] + "…"
	}
	return s
}

func telegramReason(statusCode int, tgErrorCode int, tgDescription string) string {
	// Prefer Telegram error code when present, fall back to HTTP status code.
	code := tgErrorCode
	if code == 0 {
		code = statusCode
	}

	switch code {
	case 401:
		// Telegram returns 401 for invalid/revoked tokens.
		// "Unauthorized" is too generic, so include a concrete interpretation.
		if tgDescription != "" {
			return fmt.Sprintf("unauthorized (likely invalid bot token): %q", tgDescription)
		}
		return "unauthorized (likely invalid bot token)"
	case 403:
		if tgDescription != "" {
			return fmt.Sprintf("forbidden: %q", tgDescription)
		}
		return "forbidden"
	case 429:
		if tgDescription != "" {
			return fmt.Sprintf("rate_limited: %q", tgDescription)
		}
		return "rate_limited"
	default:
		if tgDescription != "" {
			return fmt.Sprintf("telegram_error: %q", tgDescription)
		}
		return ""
	}
}

func telegramSafeMethodURL(token string, method string) string {
	method = strings.TrimPrefix(method, "/")
	if method == "" {
		method = "<unknown>"
	}
	botID := telegramBotIDFromToken(token)
	if botID == "" {
		return fmt.Sprintf("https://api.telegram.org/bot<redacted>/%s", method)
	}
	return fmt.Sprintf("https://api.telegram.org/bot%s:<redacted>/%s", botID, method)
}

func telegramBotIDFromToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	id, _, ok := strings.Cut(token, ":")
	if !ok {
		return ""
	}
	id = strings.TrimSpace(id)
	// Telegram bot id is numeric.
	for _, r := range id {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return id
}

var telegramTokenRegex = regexp.MustCompile(`^\d+:[A-Za-z0-9_-]{20,}$`)

func telegramTokenInfo(token string) string {
	env := util.GetEnvType().String()

	// Mirror util.GetBotToken() behavior to explain what likely happened,
	// without ever printing the token itself.
	_, statErr := os.Stat(util.TokenFile)
	tokenFilePresent := statErr == nil

	token = strings.TrimSpace(token)
	usingTest := token != "" && token == util.TestBotToken
	looksValid := token != "" && telegramTokenRegex.MatchString(token)

	botID := telegramBotIDFromToken(token)
	secretLen := 0
	if _, secret, ok := strings.Cut(token, ":"); ok {
		secretLen = len(secret)
	}

	parts := []string{
		fmt.Sprintf("envtype=%s", env),
	}
	if tokenFilePresent {
		parts = append(parts, fmt.Sprintf("tokenFile=%s(present)", util.TokenFile))
	} else {
		parts = append(parts, fmt.Sprintf("tokenFile=%s(missing)", util.TokenFile))
	}
	if botID != "" {
		parts = append(parts, fmt.Sprintf("botID=%s", botID))
	}
	if secretLen != 0 {
		parts = append(parts, fmt.Sprintf("secretLen=%d", secretLen))
	}
	parts = append(parts, fmt.Sprintf("looksLikeTelegramToken=%t", looksValid))
	parts = append(parts, fmt.Sprintf("usingTestToken=%t", usingTest))

	// A special case that commonly causes confusing 401s.
	if env == util.EnvTypeProd.String() && !tokenFilePresent {
		parts = append(parts, "note=prod-without-tokenFile-fallsBackToTestToken")
	}

	return strings.Join(parts, ",")
}

func telegramUnauthorizedHint(token string) string {
	// Tailor hint to this codebase: util.GetBotToken() falls back to TestBotToken in prod when bot.token is missing.
	env := util.GetEnvType()
	_, statErr := os.Stat(util.TokenFile)
	tokenFilePresent := statErr == nil

	if env == util.EnvTypeProd && !tokenFilePresent {
		return fmt.Sprintf("401 usually means bad token. In prod, missing %s makes the app fall back to TestBotToken; create %s with your real token (no trailing spaces/newlines) and restart", util.TokenFile, util.TokenFile)
	}
	if strings.TrimSpace(token) == "" {
		return "bot token is empty; ensure it is loaded correctly (bot.token or your configured source) and restart"
	}
	if token == util.TestBotToken {
		if env == util.EnvTypeProd && tokenFilePresent {
			return fmt.Sprintf("you are using TestBotToken while envtype=prod and %s is present. This usually means the token was loaded before flags/envtype were applied (init-time load) or you are reading a different bot.token due to a different working directory; ensure envtype is set at process start, bot.token contains the real token, and restart", util.TokenFile)
		}
		return "you are using TestBotToken; it may be revoked/invalid. Put a real token into bot.token (or switch envtype) and restart"
	}
	return "401 usually means the bot token is invalid/revoked or belongs to a different bot. Re-check BotFather token, ensure bot.token contains exactly the token (no spaces/newlines), and try calling getMe with curl to verify the token works"
}
