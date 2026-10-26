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

// --- admin CRUD and report -------------------------------------------------

const (
	campaignSlugMinLength         = 3
	campaignSlugMaxLength         = 40
	campaignTitleMaxLength        = 120
	campaignTargetValueMaxLength  = 255
	campaignUTMSourceMaxLength    = 64
	campaignUTMMediumMaxLength    = 64
	campaignUTMCampaignMaxLength  = 80
	campaignUTMContentMaxLength   = 80
	campaignCreatedByMaxLength    = 64
	campaignReportDefaultWindow   = 7 * 24 * time.Hour
	campaignReportDateLayout      = "2006-01-02"
	campaignReportDateTimeLayout  = "2006-01-02 15:04:05"
	campaignLinkDuplicateSlugText = "Ссылка с таким slug уже существует"
)

// campaignBotStartPattern is the subset of characters Telegram accepts in a
// start payload; the admin may only store values that survive the redirect.
var campaignBotStartPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{0,40}$`)

// CampaignLinkHandler serves the admin link CRUD and the per-link report.
type CampaignLinkHandler struct {
	db *gorm.DB
}

func NewCampaignLinkHandler(db *gorm.DB) *CampaignLinkHandler {
	return &CampaignLinkHandler{db: db}
}

// CampaignLinkWithClicks is a link row plus lifetime click totals for the
// admin list.
type CampaignLinkWithClicks struct {
	models.CampaignLink
	Clicks         int64 `json:"clicks"`
	UniqueVisitors int64 `json:"unique_visitors"`
}

// LinkReportRow is one link's traffic and money inside the report window.
// RPV is revenue per unique visitor, EPC is revenue per click.
type LinkReportRow struct {
	Slug           string  `json:"slug"`
	Title          string  `json:"title"`
	Clicks         int64   `json:"clicks"`
	UniqueVisitors int64   `json:"unique_visitors"`
	SiteVisits     int64   `json:"site_visits"`
	Leads          int64   `json:"leads"`
	Payments       int64   `json:"payments"`
	Revenue        float64 `json:"revenue"`
	RPV            float64 `json:"rpv"`
	EPC            float64 `json:"epc"`
}

// LinkReportResponse wraps the per-link rows with the global unattributed
// revenue, which by definition cannot belong to a single link.
type LinkReportResponse struct {
	Rows                []LinkReportRow `json:"rows"`
	UnattributedRevenue float64         `json:"unattributed_revenue"`
}

type createCampaignLinkRequest struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	TargetType  string `json:"target_type"`
	TargetValue string `json:"target_value"`
	UTMSource   string `json:"utm_source"`
	UTMMedium   string `json:"utm_medium"`
	UTMCampaign string `json:"utm_campaign"`
	UTMContent  string `json:"utm_content"`
	IsActive    *bool  `json:"is_active"`
}

type updateCampaignLinkRequest struct {
	Slug        *string `json:"slug"`
	Title       *string `json:"title"`
	TargetType  *string `json:"target_type"`
	TargetValue *string `json:"target_value"`
	UTMSource   *string `json:"utm_source"`
	UTMMedium   *string `json:"utm_medium"`
	UTMCampaign *string `json:"utm_campaign"`
	UTMContent  *string `json:"utm_content"`
	IsActive    *bool   `json:"is_active"`
}

// slugify lowercases and reduces raw to [a-z0-9-], collapsing separator runs
// and trimming dashes. Anything outside the allowed set acts as a separator,
// so a fully non-latin slug normalizes to "" and is rejected by validation.
func slugify(raw string) string {
	var b strings.Builder
	pendingDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if pendingDash && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingDash = false
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 {
			pendingDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if len(slug) > campaignSlugMaxLength {
		slug = strings.Trim(slug[:campaignSlugMaxLength], "-")
	}
	return slug
}

// validateCampaignLink enforces the rules the redirect handler depends on.
func validateCampaignLink(link models.CampaignLink) error {
	if len(link.Slug) < campaignSlugMinLength || len(link.Slug) > campaignSlugMaxLength {
		return errors.New("Slug должен содержать от 3 до 40 символов: строчная латиница, цифры и дефис")
	}
	for _, r := range link.Slug {
		allowed := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-'
		if !allowed {
			return errors.New("Slug может содержать только строчную латиницу, цифры и дефис")
		}
	}
	if link.Title == "" {
		return errors.New("Название ссылки обязательно")
	}
	if utf8.RuneCountInString(link.Title) > campaignTitleMaxLength {
		return errors.New("Название ссылки не должно превышать 120 символов")
	}
	if utf8.RuneCountInString(link.TargetValue) > campaignTargetValueMaxLength {
		return errors.New("Значение цели не должно превышать 255 символов")
	}
	if utf8.RuneCountInString(link.UTMSource) > campaignUTMSourceMaxLength ||
		utf8.RuneCountInString(link.UTMMedium) > campaignUTMMediumMaxLength ||
		utf8.RuneCountInString(link.UTMCampaign) > campaignUTMCampaignMaxLength ||
		utf8.RuneCountInString(link.UTMContent) > campaignUTMContentMaxLength {
		return errors.New("UTM-метки слишком длинные")
	}

	switch link.TargetType {
	case "landing":
		return nil
	case "channel":
		parsed, err := url.Parse(link.TargetValue)
		if err != nil || !campaignIsTelegramLink(link.TargetValue) || strings.Trim(parsed.Path, "/") == "" {
			return errors.New("Для канала нужна ссылка вида https://t.me/...")
		}
		return nil
	case "bot":
		if !campaignBotStartPattern.MatchString(link.TargetValue) {
			return errors.New("Для бота допустимы только латиница, цифры, _ и - (до 40 символов)")
		}
		return nil
	default:
		return errors.New("Тип цели должен быть channel, bot или landing")
	}
}

// Create stores a new campaign link.
func (h *CampaignLinkHandler) Create(c *gin.Context) {
	var req createCampaignLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	link := models.CampaignLink{
		Slug:        slugify(req.Slug),
		Title:       strings.TrimSpace(req.Title),
		TargetType:  strings.ToLower(strings.TrimSpace(req.TargetType)),
		TargetValue: strings.TrimSpace(req.TargetValue),
		UTMSource:   strings.TrimSpace(req.UTMSource),
		UTMMedium:   strings.TrimSpace(req.UTMMedium),
		UTMCampaign: strings.TrimSpace(req.UTMCampaign),
		UTMContent:  strings.TrimSpace(req.UTMContent),
		IsActive:    true,
		CreatedBy:   campaignCreatedBy(c),
	}
	desiredActive := true
	if req.IsActive != nil {
		desiredActive = *req.IsActive
	}
	if err := validateCampaignLink(link); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing models.CampaignLink
	err := h.db.Where("slug = ?", link.Slug).First(&existing).Error
	switch {
	case err == nil:
		c.JSON(http.StatusConflict, gin.H{"error": campaignLinkDuplicateSlugText})
		return
	case !errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка проверки slug"})
		return
	}

	if err := h.db.Create(&link).Error; err != nil {
		// A concurrent request may have taken the slug between the check and
		// the insert; report that as a conflict rather than a server error.
		if h.campaignLinkSlugTaken(link.Slug, 0) {
			c.JSON(http.StatusConflict, gin.H{"error": campaignLinkDuplicateSlugText})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать ссылку"})
		return
	}
	// GORM omits the zero value of a default:true bool on insert and writes the
	// default back into the struct, so an explicitly inactive link needs a
	// second, explicit write.
	if !desiredActive {
		if err := h.db.Model(&models.CampaignLink{}).Where("id = ?", link.ID).Update("is_active", false).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить статус ссылки"})
			return
		}
		link.IsActive = false
	}

	c.JSON(http.StatusCreated, link)
}

// Update changes an existing campaign link. Omitted fields keep their value.
func (h *CampaignLinkHandler) Update(c *gin.Context) {
	id, err := campaignLinkID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор ссылки"})
		return
	}

	var link models.CampaignLink
	if err := h.db.First(&link, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ссылка не найдена"})
		return
	}

	var req updateCampaignLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	candidate := link
	if req.Slug != nil {
		candidate.Slug = slugify(*req.Slug)
	}
	if req.Title != nil {
		candidate.Title = strings.TrimSpace(*req.Title)
	}
	if req.TargetType != nil {
		candidate.TargetType = strings.ToLower(strings.TrimSpace(*req.TargetType))
	}
	if req.TargetValue != nil {
		candidate.TargetValue = strings.TrimSpace(*req.TargetValue)
	}
	if req.UTMSource != nil {
		candidate.UTMSource = strings.TrimSpace(*req.UTMSource)
	}
	if req.UTMMedium != nil {
		candidate.UTMMedium = strings.TrimSpace(*req.UTMMedium)
	}
	if req.UTMCampaign != nil {
		candidate.UTMCampaign = strings.TrimSpace(*req.UTMCampaign)
	}
	if req.UTMContent != nil {
		candidate.UTMContent = strings.TrimSpace(*req.UTMContent)
	}
	if req.IsActive != nil {
		candidate.IsActive = *req.IsActive
	}

	if err := validateCampaignLink(candidate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if candidate.Slug != link.Slug && h.campaignLinkSlugTaken(candidate.Slug, link.ID) {
		c.JSON(http.StatusConflict, gin.H{"error": campaignLinkDuplicateSlugText})
		return
	}

	// A map update writes every column literally, so deactivation (false) and
	// cleared UTM values are persisted instead of being dropped as zero values.
	updates := map[string]interface{}{
		"slug":         candidate.Slug,
		"title":        candidate.Title,
		"target_type":  candidate.TargetType,
		"target_value": candidate.TargetValue,
		"utm_source":   candidate.UTMSource,
		"utm_medium":   candidate.UTMMedium,
		"utm_campaign": candidate.UTMCampaign,
		"utm_content":  candidate.UTMContent,
		"is_active":    candidate.IsActive,
	}
	if err := h.db.Model(&models.CampaignLink{}).Where("id = ?", link.ID).Updates(updates).Error; err != nil {
		if h.campaignLinkSlugTaken(candidate.Slug, link.ID) {
			c.JSON(http.StatusConflict, gin.H{"error": campaignLinkDuplicateSlugText})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить ссылку"})
		return
	}

	var saved models.CampaignLink
	if err := h.db.First(&saved, link.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось прочитать ссылку"})
		return
	}
	c.JSON(http.StatusOK, saved)
}

// Delete removes a link together with its click rows.
func (h *CampaignLinkHandler) Delete(c *gin.Context) {
	id, err := campaignLinkID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный идентификатор ссылки"})
		return
	}

	tx := h.db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления ссылки"})
		return
	}
	if err := tx.Where("link_id = ?", id).Delete(&models.CampaignLinkClick{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления переходов"})
		return
	}
	res := tx.Delete(&models.CampaignLink{}, id)
	if res.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления ссылки"})
		return
	}
	if res.RowsAffected == 0 {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Ссылка не найдена"})
		return
	}
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления ссылки"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// List returns every link with lifetime click and visitor totals.
func (h *CampaignLinkHandler) List(c *gin.Context) {
	var links []models.CampaignLink
	if err := h.db.Order("created_at DESC, id DESC").Find(&links).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения ссылок"})
		return
	}

	type clickAggregate struct {
		LinkID         uint
		Clicks         int64
		UniqueVisitors int64
	}
	var aggregates []clickAggregate
	if err := h.db.Model(&models.CampaignLinkClick{}).
		Select("link_id, COUNT(*) AS clicks, COUNT(DISTINCT NULLIF(visitor_id, '')) AS unique_visitors").
		Group("link_id").
		Scan(&aggregates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка подсчёта переходов"})
		return
	}
	byLink := make(map[uint]clickAggregate, len(aggregates))
	for _, aggregate := range aggregates {
		byLink[aggregate.LinkID] = aggregate
	}

	result := make([]CampaignLinkWithClicks, 0, len(links))
	for _, link := range links {
		aggregate := byLink[link.ID]
		result = append(result, CampaignLinkWithClicks{
			CampaignLink:   link,
			Clicks:         aggregate.Clicks,
			UniqueVisitors: aggregate.UniqueVisitors,
		})
	}
	c.JSON(http.StatusOK, result)
}

// Report aggregates clicks, site visits, leads and money per link for a
// half-open [from, to) window. Money comes only from successful payments.
func (h *CampaignLinkHandler) Report(c *gin.Context) {
	from, to, err := campaignReportWindow(c, time.Now().UTC())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	const reportQuery = `
WITH clicks AS (
    SELECT link_id,
           COUNT(*) AS clicks,
           COUNT(DISTINCT NULLIF(visitor_id, '')) AS unique_visitors
    FROM campaign_link_clicks
    WHERE created_at >= ? AND created_at < ?
    GROUP BY link_id
),
visits AS (
    SELECT c.link_id,
           COUNT(DISTINCT ae.visitor_id) AS site_visits
    FROM campaign_link_clicks c
    JOIN analytics_events ae
      ON ae.visitor_id = c.visitor_id
     AND ae.step = 'landing_view'
     AND ae.created_at >= ? AND ae.created_at < ?
    WHERE c.created_at >= ? AND c.created_at < ?
      AND c.visitor_id <> ''
    GROUP BY c.link_id
),
leads AS (
    SELECT c.link_id,
           COUNT(DISTINCT ae.visitor_id) AS leads
    FROM campaign_link_clicks c
    JOIN analytics_events ae
      ON ae.visitor_id = c.visitor_id
     AND ae.step = 'bot_lead'
     AND ae.created_at >= ? AND ae.created_at < ?
    WHERE c.created_at >= ? AND c.created_at < ?
      AND c.visitor_id <> ''
    GROUP BY c.link_id
),
paid AS (
    SELECT DISTINCT c.link_id, pt.payment_id, pt.amount
    FROM campaign_link_clicks c
    JOIN payment_transactions pt
      ON pt.visitor_id = c.visitor_id
     AND pt.status = 'SUCCESS'
     AND pt.created_at >= ? AND pt.created_at < ?
    WHERE c.created_at >= ? AND c.created_at < ?
      AND c.visitor_id <> ''
),
money AS (
    SELECT link_id,
           COUNT(*) AS payments,
           COALESCE(SUM(amount), 0) AS revenue
    FROM paid
    GROUP BY link_id
)
SELECT l.slug,
       l.title,
       COALESCE(cl.clicks, 0) AS clicks,
       COALESCE(cl.unique_visitors, 0) AS unique_visitors,
       COALESCE(v.site_visits, 0) AS site_visits,
       COALESCE(ld.leads, 0) AS leads,
       COALESCE(m.payments, 0) AS payments,
       COALESCE(m.revenue, 0) AS revenue
FROM campaign_links l
LEFT JOIN clicks cl ON cl.link_id = l.id
LEFT JOIN visits v ON v.link_id = l.id
LEFT JOIN leads ld ON ld.link_id = l.id
LEFT JOIN money m ON m.link_id = l.id
ORDER BY COALESCE(cl.clicks, 0) DESC, l.slug ASC`

	args := []interface{}{
		from, to,
		from, to, from, to,
		from, to, from, to,
		from, to, from, to,
	}
	rows := make([]LinkReportRow, 0)
	if err := h.db.Raw(reportQuery, args...).Scan(&rows).Error; err != nil {
		log.Printf("[links] failed to build the campaign link report: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка построения отчёта"})
		return
	}
	for i := range rows {
		if rows[i].Clicks > 0 {
			rows[i].EPC = rows[i].Revenue / float64(rows[i].Clicks)
		}
		if rows[i].UniqueVisitors > 0 {
			rows[i].RPV = rows[i].Revenue / float64(rows[i].UniqueVisitors)
		}
	}

	// Revenue whose payer has no click in the window belongs to no link.
	var unattributed float64
	if err := h.db.Raw(`
SELECT COALESCE(SUM(pt.amount), 0)
FROM payment_transactions pt
WHERE pt.status = 'SUCCESS'
  AND pt.created_at >= ? AND pt.created_at < ?
  AND NOT EXISTS (
      SELECT 1
      FROM campaign_link_clicks c
      WHERE c.visitor_id = pt.visitor_id
        AND c.created_at >= ? AND c.created_at < ?
        AND c.visitor_id <> ''
  )`, from, to, from, to).Scan(&unattributed).Error; err != nil {
		log.Printf("[links] failed to compute unattributed revenue: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка построения отчёта"})
		return
	}

	c.JSON(http.StatusOK, LinkReportResponse{
		Rows:                rows,
		UnattributedRevenue: unattributed,
	})
}

// campaignLinkSlugTaken reports whether another link already owns slug.
// excludeID skips the link being updated.
func (h *CampaignLinkHandler) campaignLinkSlugTaken(slug string, excludeID uint) bool {
	query := h.db.Model(&models.CampaignLink{}).Where("slug = ?", slug)
	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

// campaignLinkID parses the :id route parameter.
func campaignLinkID(c *gin.Context) (uint64, error) {
	return strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 64)
}

// campaignCreatedBy records the admin username from the JWT middleware.
func campaignCreatedBy(c *gin.Context) string {
	username := strings.TrimSpace(c.GetString("username"))
	if utf8.RuneCountInString(username) > campaignCreatedByMaxLength {
		username = string([]rune(username)[:campaignCreatedByMaxLength])
	}
	return username
}

// campaignReportWindow parses from/to, defaulting to the last seven days.
// Accepted inputs are RFC3339, YYYY-MM-DD and YYYY-MM-DD HH:MM:SS (UTC).
func campaignReportWindow(c *gin.Context, now time.Time) (time.Time, time.Time, error) {
	to := now
	if raw := strings.TrimSpace(c.Query("to")); raw != "" {
		parsed, err := parseCampaignReportTime(raw)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("Параметр to должен быть датой (YYYY-MM-DD) или RFC3339")
		}
		to = parsed
	}
	from := to.Add(-campaignReportDefaultWindow)
	if raw := strings.TrimSpace(c.Query("from")); raw != "" {
		parsed, err := parseCampaignReportTime(raw)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("Параметр from должен быть датой (YYYY-MM-DD) или RFC3339")
		}
		from = parsed
	}
	if !from.Before(to) {
		return time.Time{}, time.Time{}, errors.New("Начало периода должно быть раньше конца")
	}
	return from, to, nil
}

func parseCampaignReportTime(raw string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, campaignReportDateLayout, campaignReportDateTimeLayout} {
		if parsed, err := time.ParseInLocation(layout, raw, time.UTC); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, errors.New("unsupported time format")
}
