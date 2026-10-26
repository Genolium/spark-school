package handlers

import (
	"crypto/rand"
	"errors"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
)

const (
	// campaignBotURL is the only bot the redirect may send visitors to.
	campaignBotURL = "https://t.me/spark_prep_bot"
	// campaignTelegramHost is the only host allowed for channel links.
	campaignTelegramHost = "t.me"

	campaignVisitorIDLength     = 20
	campaignAttributionIDLength = 12
	// campaignVisitorCookieMaxAge is one year, matching the site's cookie.
	campaignVisitorCookieMaxAge = 60 * 60 * 24 * 365
	// Telegram caps the start payload at 64 characters.
	campaignStartParamMaxLength = 64
	// campaign_links_clicks.user_agent and referer are VARCHAR(255).
	campaignHeaderMaxBytes = 255
)

const (
	// campaignVisitorCookieName is the first-party cookie shared with the site
	// so repeat /r clicks and later page views reuse one visitor id.
	campaignVisitorCookieName = "spark_vid"
)

var (
	campaignVisitorIDPattern     = regexp.MustCompile(`^v[a-z0-9]{6,40}$`)
	campaignAttributionIDPattern = regexp.MustCompile(`^a[a-z0-9]{6,40}$`)
	campaignStartValueSanitizer  = regexp.MustCompile(`[^A-Za-z0-9_-]`)
)

// RedirectCampaignLink serves public /r/:slug links. It records one anonymous
// click and redirects to the channel, the bot or the landing page with campaign
// UTMs. Click recording is best-effort: analytics must never break the redirect.
func RedirectCampaignLink(db *gorm.DB, frontendURL, fallbackChannelURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := strings.TrimSpace(c.Param("slug"))
		if slug == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Ссылка не найдена"})
			return
		}

		var link models.CampaignLink
		if err := db.Where("slug = ? AND is_active = ?", slug, true).First(&link).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("[links] failed to load campaign link %q: %v", slug, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Внутренняя ошибка"})
				return
			}
			c.JSON(http.StatusNotFound, gin.H{"error": "Ссылка не найдена"})
			return
		}

		visitorID := resolveCampaignVisitorID(c)
		attributionToken := campaignExistingID(c.Query("attribution_token"), campaignAttributionIDPattern)
		if attributionToken == "" {
			attributionToken = campaignRandomID("a", campaignAttributionIDLength)
		}

		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(
			campaignVisitorCookieName,
			visitorID,
			campaignVisitorCookieMaxAge,
			"/",
			"",
			campaignCookieSecure(c),
			false,
		)

		recordCampaignClick(db, link, visitorID, attributionToken, c)

		c.Header("Cache-Control", "no-store")
		c.Redirect(http.StatusFound, campaignTargetURL(link, visitorID, attributionToken, frontendURL, fallbackChannelURL))
	}
}

// campaignTargetURL builds the redirect destination. A channel target is only
// honoured for t.me links so the endpoint can never become an open redirect.
func campaignTargetURL(link models.CampaignLink, visitorID, attributionToken, frontendURL, fallbackChannelURL string) string {
	base := strings.TrimRight(strings.TrimSpace(frontendURL), "/")

	switch strings.ToLower(strings.TrimSpace(link.TargetType)) {
	case "bot":
		return campaignBotTargetURL(link.TargetValue, attributionToken)
	case "channel":
		target := strings.TrimSpace(link.TargetValue)
		if campaignIsTelegramLink(target) {
			return target
		}
		fallback := strings.TrimSpace(fallbackChannelURL)
		if fallback == "" {
			fallback = campaignBotURL
		}
		return fallback
	case "landing":
		if base == "" {
			return "/"
		}
		query := url.Values{}
		if visitorID != "" {
			query.Set("spark_vid", visitorID)
		}
		addUTM := func(key, value string) {
			if value = strings.TrimSpace(value); value != "" {
				query.Set(key, value)
			}
		}
		addUTM("utm_source", link.UTMSource)
		addUTM("utm_medium", link.UTMMedium)
		addUTM("utm_campaign", link.UTMCampaign)
		addUTM("utm_content", link.UTMContent)
		if encoded := query.Encode(); encoded != "" {
			return base + "/?" + encoded
		}
		return base + "/"
	default:
		// Unknown target types are a data error; send the visitor somewhere safe.
		if base == "" {
			return "/"
		}
		return base + "/"
	}
}

// campaignIsTelegramLink reports whether target is an https link to t.me.
// The host is compared case-insensitively and parsed rather than prefix-matched
// so lookalike hosts (t.me.evil.example, t.me@evil.example) are rejected.
func campaignIsTelegramLink(target string) bool {
	parsed, err := url.Parse(target)
	if err != nil {
		return false
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return false
	}
	return strings.EqualFold(parsed.Hostname(), campaignTelegramHost)
}

// campaignBotTargetURL builds a bot deep link whose start payload carries the
// target value and the attribution token, trimmed to Telegram's limit. The
// action is shortened first so the attribution token always survives intact.
func campaignBotTargetURL(targetValue, attributionToken string) string {
	action := campaignStartValueSanitizer.ReplaceAllString(strings.TrimSpace(targetValue), "")
	var start string
	if attributionToken == "" {
		start = action
	} else if action == "" {
		start = "a_" + attributionToken
	} else {
		suffix := "_a_" + attributionToken
		maxAction := campaignStartParamMaxLength - len(suffix)
		if maxAction < 0 {
			maxAction = 0
		}
		if len(action) > maxAction {
			action = action[:maxAction]
		}
		start = action + suffix
	}
	if len(start) > campaignStartParamMaxLength {
		start = start[:campaignStartParamMaxLength]
	}
	if start == "" {
		return campaignBotURL
	}
	return campaignBotURL + "?start=" + url.QueryEscape(start)
}

// recordCampaignClick stores the click without ever persisting a raw IP.
func recordCampaignClick(db *gorm.DB, link models.CampaignLink, visitorID, attributionToken string, c *gin.Context) {
	click := models.CampaignLinkClick{
		LinkID:           link.ID,
		VisitorID:        visitorID,
		AttributionToken: attributionToken,
		UserAgent:        campaignTruncateHeader(c.GetHeader("User-Agent")),
		Referer:          campaignTruncateHeader(c.GetHeader("Referer")),
	}
	if err := db.Create(&click).Error; err != nil {
		log.Printf("[links] failed to record click for slug %q: %v", link.Slug, err)
	}
}

// resolveCampaignVisitorID reuses the id the caller already carries, preferring
// the documented ?visitor_id= contract and then the shared spark_vid cookie.
// Only well-formed ids are trusted; anything else is replaced with a fresh one.
func resolveCampaignVisitorID(c *gin.Context) string {
	if id := campaignExistingID(c.Query("visitor_id"), campaignVisitorIDPattern); id != "" {
		return id
	}
	if cookie, err := c.Cookie(campaignVisitorCookieName); err == nil {
		if id := campaignExistingID(cookie, campaignVisitorIDPattern); id != "" {
			return id
		}
	}
	return campaignRandomID("v", campaignVisitorIDLength)
}

// campaignCookieSecure marks the visitor cookie Secure only on TLS requests,
// including TLS terminated by the reverse proxy.
func campaignCookieSecure(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")), "https")
}

// campaignExistingID accepts an id supplied by the caller only when it matches
// the documented format, so the redirect cannot inject arbitrary tokens.
func campaignExistingID(value string, pattern *regexp.Regexp) string {
	value = strings.TrimSpace(value)
	if !pattern.MatchString(value) {
		return ""
	}
	return value
}

// campaignTruncateHeader trims an HTTP header to the column width without
// cutting a multi-byte rune in half.
func campaignTruncateHeader(value string) string {
	if len(value) <= campaignHeaderMaxBytes {
		return value
	}
	value = value[:campaignHeaderMaxBytes]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}

// campaignRandomID returns prefix + n lowercase alphanumeric characters from
// crypto/rand. The timestamp fallback only runs if the system CSPRNG fails.
func campaignRandomID(prefix string, n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		log.Printf("[links] crypto/rand unavailable, using timestamp id: %v", err)
		timestamp := strconv.FormatInt(time.Now().UnixNano(), 36)
		timestamp = strings.Repeat("0", n) + timestamp
		return prefix + timestamp[len(timestamp)-n:]
	}
	var b strings.Builder
	b.Grow(n + len(prefix))
	b.WriteString(prefix)
	for _, v := range buf {
		b.WriteByte(alphabet[int(v)%len(alphabet)])
	}
	return b.String()
}
