package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	testLinksFrontendURL = "https://spark.example"
	testLinksChannelURL  = "https://t.me/spark_prep_channel"
)

var (
	testVisitorIDPattern     = regexp.MustCompile(`^v[a-z0-9]{20}$`)
	testAttributionIDPattern = regexp.MustCompile(`^a[a-z0-9]{12}$`)
)

// linksPGDB prepares an isolated PostgreSQL schema for the campaign-link
// redirect tests. It skips when TEST_DATABASE_URL is not set, matching the
// other PostgreSQL integration suites in this package.
func linksPGDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}
	lockPath := filepath.Join(os.TempDir(), "spark-pg-integration.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	})

	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.Exec("DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;"); err != nil {
		t.Fatal(err)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.CampaignLink{}, &models.CampaignLinkClick{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func newCampaignLinksRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/r/:slug", RedirectCampaignLink(db, testLinksFrontendURL, testLinksChannelURL))
	router.HEAD("/r/:slug", RedirectCampaignLink(db, testLinksFrontendURL, testLinksChannelURL))
	return router
}

func seedCampaignLink(t *testing.T, db *gorm.DB, link *models.CampaignLink) {
	t.Helper()
	if err := db.Create(link).Error; err != nil {
		t.Fatalf("failed to seed campaign link: %v", err)
	}
}

func getCampaignLink(router *gin.Engine, slug string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/r/"+slug, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Test)")
	req.Header.Set("Referer", "https://youtube.com/watch")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func getCampaignLinkWithVisitorCookie(router *gin.Engine, slug, cookieValue string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/r/"+slug, nil)
	req.AddCookie(&http.Cookie{Name: "spark_vid", Value: cookieValue})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func setVisitorCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "spark_vid" {
			return cookie
		}
	}
	t.Fatalf("expected the redirect to set a spark_vid cookie, headers: %v", rec.Header())
	return nil
}

func TestRedirectCampaignLink_RecordsClickAndRedirects(t *testing.T) {
	db := linksPGDB(t)
	link := models.CampaignLink{
		Slug:        "yt",
		Title:       "YouTube october",
		TargetType:  "landing",
		UTMSource:   "youtube",
		UTMCampaign: "oct",
		IsActive:    true,
	}
	seedCampaignLink(t, db, &link)
	router := newCampaignLinksRouter(db)

	rec := getCampaignLink(router, "yt")

	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d (body %q)", http.StatusFound, rec.Code, rec.Body.String())
	}
	location, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("invalid Location header: %v", err)
	}
	if location.Scheme != "https" || location.Host != "spark.example" {
		t.Fatalf("expected redirect to the frontend, got %q", location.String())
	}
	query := location.Query()
	if got := query.Get("utm_source"); got != "youtube" {
		t.Errorf("expected utm_source=youtube, got %q", got)
	}
	if got := query.Get("utm_campaign"); got != "oct" {
		t.Errorf("expected utm_campaign=oct, got %q", got)
	}
	if strings.Contains(location.RawQuery, "utm_medium=") || strings.Contains(location.RawQuery, "utm_content=") {
		t.Errorf("empty utm parameters must be omitted, got %q", location.RawQuery)
	}
	visitorID := query.Get("spark_vid")
	if !testVisitorIDPattern.MatchString(visitorID) {
		t.Fatalf("expected spark_vid to match %s, got %q", testVisitorIDPattern, visitorID)
	}

	var clicks []models.CampaignLinkClick
	if err := db.Order("id asc").Find(&clicks).Error; err != nil {
		t.Fatalf("failed to load clicks: %v", err)
	}
	if len(clicks) != 1 {
		t.Fatalf("expected exactly one recorded click, got %d", len(clicks))
	}
	click := clicks[0]
	if click.LinkID != link.ID {
		t.Errorf("expected click to reference link %d, got %d", link.ID, click.LinkID)
	}
	if click.VisitorID != visitorID {
		t.Errorf("expected click visitor id %q, got %q", visitorID, click.VisitorID)
	}
	if !testAttributionIDPattern.MatchString(click.AttributionToken) {
		t.Errorf("expected attribution token to match %s, got %q", testAttributionIDPattern, click.AttributionToken)
	}
	if click.IPHash != "" {
		t.Errorf("raw IP must not be stored, got ip_hash %q", click.IPHash)
	}
	if click.UserAgent != "Mozilla/5.0 (Test)" {
		t.Errorf("expected user agent to be recorded, got %q", click.UserAgent)
	}
	if click.Referer != "https://youtube.com/watch" {
		t.Errorf("expected referer to be recorded, got %q", click.Referer)
	}
}

func TestRedirectCampaignLink_HeadRequestRedirects(t *testing.T) {
	db := linksPGDB(t)
	seedCampaignLink(t, db, &models.CampaignLink{
		Slug:       "yt",
		Title:      "YouTube",
		TargetType: "landing",
		IsActive:   true,
	})
	router := newCampaignLinksRouter(db)

	req := httptest.NewRequest(http.MethodHead, "/r/yt", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d for HEAD, got %d", http.StatusFound, rec.Code)
	}
	if rec.Header().Get("Location") == "" {
		t.Fatalf("expected a Location header for HEAD")
	}
}

func TestRedirectCampaignLink_ReusesVisitorCookie(t *testing.T) {
	db := linksPGDB(t)
	seedCampaignLink(t, db, &models.CampaignLink{
		Slug:       "yt",
		Title:      "YouTube",
		TargetType: "landing",
		IsActive:   true,
	})
	router := newCampaignLinksRouter(db)

	const visitorID = "va1b2c3d4e5f6g7h8i9j0"
	for i := 0; i < 2; i++ {
		rec := getCampaignLinkWithVisitorCookie(router, "yt", visitorID)
		if rec.Code != http.StatusFound {
			t.Fatalf("request %d: expected status %d, got %d", i+1, http.StatusFound, rec.Code)
		}
		location, err := url.Parse(rec.Header().Get("Location"))
		if err != nil {
			t.Fatalf("request %d: invalid Location header: %v", i+1, err)
		}
		if got := location.Query().Get("spark_vid"); got != visitorID {
			t.Errorf("request %d: expected the cookie id in Location, got %q", i+1, got)
		}
		if got := setVisitorCookie(t, rec).Value; got != visitorID {
			t.Errorf("request %d: expected the cookie to be refreshed with %q, got %q", i+1, visitorID, got)
		}
	}

	var clicks []models.CampaignLinkClick
	if err := db.Order("id asc").Find(&clicks).Error; err != nil {
		t.Fatalf("failed to load clicks: %v", err)
	}
	if len(clicks) != 2 {
		t.Fatalf("expected two click rows, got %d", len(clicks))
	}
	for i, click := range clicks {
		if click.VisitorID != visitorID {
			t.Errorf("click %d: expected visitor id %q, got %q", i+1, visitorID, click.VisitorID)
		}
	}
}

func TestRedirectCampaignLink_SetsVisitorCookie(t *testing.T) {
	db := linksPGDB(t)
	seedCampaignLink(t, db, &models.CampaignLink{
		Slug:       "yt",
		Title:      "YouTube",
		TargetType: "landing",
		IsActive:   true,
	})
	router := newCampaignLinksRouter(db)

	rec := getCampaignLink(router, "yt")
	cookie := setVisitorCookie(t, rec)
	if !testVisitorIDPattern.MatchString(cookie.Value) {
		t.Fatalf("expected a minted visitor id, got %q", cookie.Value)
	}
	if cookie.Path != "/" {
		t.Errorf("expected cookie Path=/, got %q", cookie.Path)
	}
	if cookie.MaxAge != 31536000 {
		t.Errorf("expected cookie MaxAge=31536000, got %d", cookie.MaxAge)
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("expected SameSite=Lax, got %v", cookie.SameSite)
	}
	if cookie.HttpOnly {
		t.Errorf("expected HttpOnly=false so the site JS can read the cookie")
	}
	if cookie.Secure {
		t.Errorf("expected Secure=false on a plain HTTP request")
	}

	location, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("invalid Location header: %v", err)
	}
	if got := location.Query().Get("spark_vid"); got != cookie.Value {
		t.Errorf("expected Location spark_vid %q, got %q", cookie.Value, got)
	}

	var click models.CampaignLinkClick
	if err := db.First(&click).Error; err != nil {
		t.Fatalf("expected a recorded click, got %v", err)
	}
	if click.VisitorID != cookie.Value {
		t.Errorf("expected the click to use the cookie id %q, got %q", cookie.Value, click.VisitorID)
	}
}

func TestRedirectCampaignLink_ReplacesInvalidVisitorCookie(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{name: "uuid", value: "8f14e45f-ea3b-4c1d-9e2f-3a4b5c6d7e8f"},
		{name: "uppercase", value: "VABCDEF1234567890ABCDE"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := linksPGDB(t)
			seedCampaignLink(t, db, &models.CampaignLink{
				Slug:       "yt",
				Title:      "YouTube",
				TargetType: "landing",
				IsActive:   true,
			})
			router := newCampaignLinksRouter(db)

			rec := getCampaignLinkWithVisitorCookie(router, "yt", tc.value)
			cookie := setVisitorCookie(t, rec)
			if cookie.Value == tc.value {
				t.Fatalf("expected the invalid cookie %q to be replaced", tc.value)
			}
			if !testVisitorIDPattern.MatchString(cookie.Value) {
				t.Fatalf("expected a minted visitor id, got %q", cookie.Value)
			}

			var click models.CampaignLinkClick
			if err := db.First(&click).Error; err != nil {
				t.Fatalf("expected a recorded click, got %v", err)
			}
			if click.VisitorID != cookie.Value {
				t.Errorf("expected the click to use the replacement id %q, got %q", cookie.Value, click.VisitorID)
			}
		})
	}
}

func TestRedirectCampaignLink_VisitorCookieSecureOnTLS(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*http.Request)
	}{
		{
			name:  "tls connection",
			setup: func(*http.Request) {},
		},
		{
			name: "proxy terminated tls",
			setup: func(req *http.Request) {
				req.Header.Set("X-Forwarded-Proto", "https")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := linksPGDB(t)
			seedCampaignLink(t, db, &models.CampaignLink{
				Slug:       "yt",
				Title:      "YouTube",
				TargetType: "landing",
				IsActive:   true,
			})
			router := newCampaignLinksRouter(db)

			req := httptest.NewRequest(http.MethodGet, "https://spark.example/r/yt", nil)
			tc.setup(req)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusFound {
				t.Fatalf("expected status %d, got %d", http.StatusFound, rec.Code)
			}
			if !setVisitorCookie(t, rec).Secure {
				t.Fatalf("expected the visitor cookie to be Secure")
			}
		})
	}
}

func TestRedirectCampaignLink_UnknownSlugIs404(t *testing.T) {
	db := linksPGDB(t)
	router := newCampaignLinksRouter(db)

	rec := getCampaignLink(router, "missing")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}

	var count int64
	if err := db.Model(&models.CampaignLinkClick{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count clicks: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no click rows for an unknown slug, got %d", count)
	}
}

func TestRedirectCampaignLink_InactiveSlugIs404(t *testing.T) {
	db := linksPGDB(t)
	link := models.CampaignLink{
		Slug:       "off",
		Title:      "Disabled",
		TargetType: "landing",
		IsActive:   true,
	}
	seedCampaignLink(t, db, &link)
	// GORM omits the zero value on insert and lets the column default (true)
	// win, so deactivate the row explicitly.
	if err := db.Model(&models.CampaignLink{}).Where("slug = ?", "off").Update("is_active", false).Error; err != nil {
		t.Fatalf("failed to deactivate link: %v", err)
	}
	router := newCampaignLinksRouter(db)

	rec := getCampaignLink(router, "off")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d for an inactive link, got %d", http.StatusNotFound, rec.Code)
	}

	var count int64
	if err := db.Model(&models.CampaignLinkClick{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count clicks: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no click rows for an inactive slug, got %d", count)
	}
}

func TestRedirectCampaignLink_BotTargetCarriesAttribution(t *testing.T) {
	db := linksPGDB(t)
	seedCampaignLink(t, db, &models.CampaignLink{
		Slug:        "yt",
		Title:       "YouTube bot",
		TargetType:  "bot",
		TargetValue: "src_youtube",
		IsActive:    true,
	})
	router := newCampaignLinksRouter(db)

	rec := getCampaignLink(router, "yt")
	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d (body %q)", http.StatusFound, rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "t.me/spark_prep_bot?start=") {
		t.Fatalf("expected a bot deep link, got %q", location)
	}
	if !strings.Contains(location, "src_youtube") {
		t.Errorf("expected the start payload to carry the target value, got %q", location)
	}
	if !strings.Contains(location, "_a_") {
		t.Errorf("expected the start payload to carry the attribution token, got %q", location)
	}
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("invalid Location header: %v", err)
	}
	start := parsed.Query().Get("start")
	if len(start) > 64 {
		t.Errorf("Telegram start payload must be trimmed to 64 characters, got %d (%q)", len(start), start)
	}
	if !strings.HasPrefix(start, "src_youtube_a_a") {
		t.Errorf("expected start payload to begin with src_youtube_a_a, got %q", start)
	}
}

func TestRedirectCampaignLink_TruncatesOversizedHeaders(t *testing.T) {
	db := linksPGDB(t)
	seedCampaignLink(t, db, &models.CampaignLink{
		Slug:       "yt",
		Title:      "YouTube",
		TargetType: "landing",
		IsActive:   true,
	})
	router := newCampaignLinksRouter(db)

	longAgent := strings.Repeat("a", 300)
	longReferer := strings.Repeat("ж", 200) // 400 bytes, 200 runes
	req := httptest.NewRequest(http.MethodGet, "/r/yt", nil)
	req.Header.Set("User-Agent", longAgent)
	req.Header.Set("Referer", longReferer)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, rec.Code)
	}
	var click models.CampaignLinkClick
	if err := db.First(&click).Error; err != nil {
		t.Fatalf("expected a recorded click, got %v", err)
	}
	if click.UserAgent != longAgent[:255] {
		t.Errorf("expected the user agent to be cut to 255 bytes, got %d bytes", len(click.UserAgent))
	}
	if len(click.Referer) > 255 {
		t.Errorf("expected referer to stay within 255 bytes, got %d", len(click.Referer))
	}
	if !utf8.ValidString(click.Referer) {
		t.Errorf("expected a valid UTF-8 referer after truncation")
	}
}

func TestRedirectCampaignLink_BotTargetWithoutTargetValue(t *testing.T) {
	db := linksPGDB(t)
	seedCampaignLink(t, db, &models.CampaignLink{
		Slug:       "yt",
		Title:      "YouTube bot",
		TargetType: "bot",
		IsActive:   true,
	})
	router := newCampaignLinksRouter(db)

	rec := getCampaignLink(router, "yt")
	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, rec.Code)
	}
	parsed, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("invalid Location header: %v", err)
	}
	start := parsed.Query().Get("start")
	if !strings.HasPrefix(start, "a_a") {
		t.Fatalf("expected the start payload to be a_<attribution>, got %q", start)
	}
	if len(start) > 64 {
		t.Fatalf("expected the start payload to stay within 64 characters, got %d", len(start))
	}
}

func TestRedirectCampaignLink_LongBotTargetKeepsAttribution(t *testing.T) {
	db := linksPGDB(t)
	longAction := strings.Repeat("x", 60)
	seedCampaignLink(t, db, &models.CampaignLink{
		Slug:        "yt",
		Title:       "YouTube bot",
		TargetType:  "bot",
		TargetValue: longAction,
		IsActive:    true,
	})
	router := newCampaignLinksRouter(db)

	rec := getCampaignLink(router, "yt")
	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, rec.Code)
	}
	parsed, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("invalid Location header: %v", err)
	}
	start := parsed.Query().Get("start")
	if len(start) > 64 {
		t.Fatalf("expected the start payload to stay within 64 characters, got %d", len(start))
	}
	parts := strings.Split(start, "_a_")
	if len(parts) != 2 {
		t.Fatalf("expected the start payload to carry exactly one _a_ separator, got %q", start)
	}
	if parts[0] != longAction[:64-3-13] {
		t.Errorf("expected the action to be shortened to %d characters, got %q", 64-3-13, parts[0])
	}
	if !testAttributionIDPattern.MatchString(parts[1]) {
		t.Fatalf("expected an intact attribution token, got %q", parts[1])
	}

	var click models.CampaignLinkClick
	if err := db.First(&click).Error; err != nil {
		t.Fatalf("expected a recorded click, got %v", err)
	}
	if click.AttributionToken != parts[1] {
		t.Fatalf("expected the payload token to match the recorded attribution token, payload %q click %q", parts[1], click.AttributionToken)
	}
}

func TestRedirectCampaignLink_DoesNotOpenRedirect(t *testing.T) {
	cases := []struct {
		name        string
		targetValue string
		want        string
	}{
		{
			name:        "external http target falls back to the channel",
			targetValue: "https://evil.example/phish",
			want:        testLinksChannelURL,
		},
		{
			name:        "lookalike telegram host falls back to the channel",
			targetValue: "https://t.me.evil.example/phish",
			want:        testLinksChannelURL,
		},
		{
			name:        "valid telegram target is used",
			targetValue: "https://t.me/spark_prep_channel/42",
			want:        "https://t.me/spark_prep_channel/42",
		},
		{
			name:        "telegram host is matched case-insensitively",
			targetValue: "https://T.ME/spark_prep_channel/7",
			want:        "https://T.ME/spark_prep_channel/7",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := linksPGDB(t)
			seedCampaignLink(t, db, &models.CampaignLink{
				Slug:        "ch",
				Title:       "Channel",
				TargetType:  "channel",
				TargetValue: tc.targetValue,
				IsActive:    true,
			})
			router := newCampaignLinksRouter(db)

			rec := getCampaignLink(router, "ch")
			if rec.Code != http.StatusFound {
				t.Fatalf("expected status %d, got %d (body %q)", http.StatusFound, rec.Code, rec.Body.String())
			}
			if got := rec.Header().Get("Location"); got != tc.want {
				t.Fatalf("expected Location %q, got %q", tc.want, got)
			}
			if strings.Contains(rec.Header().Get("Location"), "evil.example") {
				t.Fatalf("open redirect is not allowed, got %q", rec.Header().Get("Location"))
			}
		})
	}
}

// --- admin CRUD and report -------------------------------------------------

// linksAdminPGDB prepares the campaign-link tables plus the analytics and
// payment tables the report joins against.
func linksAdminPGDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := linksPGDB(t)
	if err := db.AutoMigrate(&models.AnalyticsEvent{}, &models.PaymentTransaction{}); err != nil {
		t.Fatalf("failed to migrate analytics/payment tables: %v", err)
	}
	return db
}

func newCampaignLinkAdminRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewCampaignLinkHandler(db)
	admin := router.Group("/api/v1/admin")
	admin.GET("/links", handler.List)
	admin.POST("/links", handler.Create)
	admin.PUT("/links/:id", handler.Update)
	admin.DELETE("/links/:id", handler.Delete)
	admin.GET("/links/report", handler.Report)
	return router
}

func performJSON(t *testing.T, router *gin.Engine, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	var body []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("failed to encode request: %v", err)
		}
		body = encoded
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("failed to decode response %q: %v", rec.Body.String(), err)
	}
	return out
}

func expectStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("expected status %d, got %d (body %q)", want, rec.Code, rec.Body.String())
	}
}

func assertFloat(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.005 {
		t.Errorf("expected %s %.4f, got %.4f", name, want, got)
	}
}

func TestSlugify(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{name: "lowercases and hyphenates", raw: "YouTube Oct!", want: "youtube-oct"},
		{name: "collapses separators", raw: "  --Hello__World--  ", want: "hello-world"},
		{name: "drops dots to a separator", raw: "a.b", want: "a-b"},
		{name: "drops query and slash noise", raw: "/r/Spring?x=1", want: "r-spring-x-1"},
		{name: "keeps digits", raw: "Mixed CASE 42", want: "mixed-case-42"},
		{name: "drops non-latin script", raw: "уже-латиница", want: ""},
		{name: "truncates to forty characters", raw: strings.Repeat("x", 60), want: strings.Repeat("x", 40)},
		{name: "trims a trailing dash after truncation", raw: strings.Repeat("a", 39) + " b", want: strings.Repeat("a", 39)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := slugify(tc.raw); got != tc.want {
				t.Fatalf("slugify(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestCampaignLinkAdmin_CreateNormalizesInput(t *testing.T) {
	db := linksAdminPGDB(t)
	router := newCampaignLinkAdminRouter(db)

	rec := performJSON(t, router, http.MethodPost, "/api/v1/admin/links", map[string]any{
		"slug":        "  YouTube-Oct_2026!! ",
		"title":       "  YouTube October  ",
		"target_type": " Landing ",
		"utm_source":  "  youtube ",
	})
	expectStatus(t, rec, http.StatusCreated)

	created := decodeJSON[models.CampaignLink](t, rec)
	if created.ID == 0 {
		t.Fatalf("expected the created link to carry an id")
	}
	if created.Slug != "youtube-oct-2026" {
		t.Errorf("expected a normalized slug, got %q", created.Slug)
	}
	if created.Title != "YouTube October" {
		t.Errorf("expected a trimmed title, got %q", created.Title)
	}
	if created.TargetType != "landing" {
		t.Errorf("expected a normalized target type, got %q", created.TargetType)
	}
	if created.UTMSource != "youtube" {
		t.Errorf("expected a trimmed utm_source, got %q", created.UTMSource)
	}
	if !created.IsActive {
		t.Errorf("expected a new link to be active")
	}

	var stored models.CampaignLink
	if err := db.First(&stored, created.ID).Error; err != nil {
		t.Fatalf("expected the link in the database: %v", err)
	}
	if stored.Slug != "youtube-oct-2026" || !stored.IsActive {
		t.Errorf("unexpected stored link: %+v", stored)
	}

	botRec := performJSON(t, router, http.MethodPost, "/api/v1/admin/links", map[string]any{
		"slug":         "bot-link",
		"title":        "Bot link",
		"target_type":  "bot",
		"target_value": "src_youtube",
	})
	expectStatus(t, botRec, http.StatusCreated)

	emptyBotRec := performJSON(t, router, http.MethodPost, "/api/v1/admin/links", map[string]any{
		"slug":        "bot-empty",
		"title":       "Bot link without payload",
		"target_type": "bot",
	})
	expectStatus(t, emptyBotRec, http.StatusCreated)
}

func TestCampaignLinkAdmin_CreateRejectsInvalidInput(t *testing.T) {
	db := linksAdminPGDB(t)
	router := newCampaignLinkAdminRouter(db)

	base := func() map[string]any {
		return map[string]any{
			"slug":        "valid-slug",
			"title":       "Valid title",
			"target_type": "landing",
		}
	}

	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "slug without allowed characters", mutate: func(m map[string]any) { m["slug"] = "!!!" }},
		{name: "slug too short", mutate: func(m map[string]any) { m["slug"] = "ab" }},
		{name: "missing slug", mutate: func(m map[string]any) { delete(m, "slug") }},
		{name: "missing title", mutate: func(m map[string]any) { m["title"] = "   " }},
		{name: "title too long", mutate: func(m map[string]any) { m["title"] = strings.Repeat("т", 121) }},
		{name: "unknown target type", mutate: func(m map[string]any) { m["target_type"] = "email" }},
		{name: "missing target type", mutate: func(m map[string]any) { delete(m, "target_type") }},
		{name: "channel with a foreign host", mutate: func(m map[string]any) {
			m["target_type"] = "channel"
			m["target_value"] = "https://evil.example/phish"
		}},
		{name: "channel lookalike host", mutate: func(m map[string]any) {
			m["target_type"] = "channel"
			m["target_value"] = "https://t.me.evil.example/phish"
		}},
		{name: "channel without tls", mutate: func(m map[string]any) {
			m["target_type"] = "channel"
			m["target_value"] = "http://t.me/spark_prep_channel"
		}},
		{name: "channel without a path", mutate: func(m map[string]any) {
			m["target_type"] = "channel"
			m["target_value"] = "https://t.me/"
		}},
		{name: "bot payload with spaces", mutate: func(m map[string]any) {
			m["target_type"] = "bot"
			m["target_value"] = "bad payload!"
		}},
		{name: "bot payload too long", mutate: func(m map[string]any) {
			m["target_type"] = "bot"
			m["target_value"] = strings.Repeat("a", 41)
		}},
		{name: "utm_source too long", mutate: func(m map[string]any) { m["utm_source"] = strings.Repeat("u", 65) }},
		{name: "utm_campaign too long", mutate: func(m map[string]any) { m["utm_campaign"] = strings.Repeat("c", 81) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := base()
			tc.mutate(payload)
			rec := performJSON(t, router, http.MethodPost, "/api/v1/admin/links", payload)
			expectStatus(t, rec, http.StatusBadRequest)
		})
	}
}

func TestCampaignLinkAdmin_CreateRejectsDuplicateSlug(t *testing.T) {
	db := linksAdminPGDB(t)
	router := newCampaignLinkAdminRouter(db)

	first := performJSON(t, router, http.MethodPost, "/api/v1/admin/links", map[string]any{
		"slug":        "ytb",
		"title":       "YouTube",
		"target_type": "landing",
	})
	expectStatus(t, first, http.StatusCreated)

	// The second slug normalizes to the same value, so it must collide even
	// though the request spells it differently.
	second := performJSON(t, router, http.MethodPost, "/api/v1/admin/links", map[string]any{
		"slug":        "  YTB  ",
		"title":       "YouTube duplicate",
		"target_type": "landing",
	})
	expectStatus(t, second, http.StatusConflict)

	var count int64
	if err := db.Model(&models.CampaignLink{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count links: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one link after a conflict, got %d", count)
	}
}

func TestCampaignLinkAdmin_UpdateAndDeactivate(t *testing.T) {
	db := linksAdminPGDB(t)
	router := newCampaignLinkAdminRouter(db)

	rec := performJSON(t, router, http.MethodPost, "/api/v1/admin/links", map[string]any{
		"slug":        "ytb",
		"title":       "YouTube",
		"target_type": "landing",
		"utm_source":  "youtube",
	})
	expectStatus(t, rec, http.StatusCreated)
	link := decodeJSON[models.CampaignLink](t, rec)

	// The bool column has default:true, so a struct update would silently drop
	// false. The handler must persist an explicit deactivation.
	deactivate := performJSON(t, router, http.MethodPut, "/api/v1/admin/links/"+strconv.FormatUint(uint64(link.ID), 10), map[string]any{
		"title":     "YouTube (paused)",
		"is_active": false,
	})
	expectStatus(t, deactivate, http.StatusOK)
	updated := decodeJSON[models.CampaignLink](t, deactivate)
	if updated.IsActive {
		t.Errorf("expected the response to report an inactive link")
	}
	if updated.Title != "YouTube (paused)" {
		t.Errorf("expected the title to change, got %q", updated.Title)
	}
	if updated.Slug != "ytb" || updated.UTMSource != "youtube" {
		t.Errorf("omitted fields must be preserved, got %+v", updated)
	}

	var stored models.CampaignLink
	if err := db.First(&stored, link.ID).Error; err != nil {
		t.Fatalf("failed to reload the link: %v", err)
	}
	if stored.IsActive {
		t.Fatalf("expected is_active=false in the database, got true")
	}

	activate := performJSON(t, router, http.MethodPut, "/api/v1/admin/links/"+strconv.FormatUint(uint64(link.ID), 10), map[string]any{
		"is_active": true,
	})
	expectStatus(t, activate, http.StatusOK)
	if !decodeJSON[models.CampaignLink](t, activate).IsActive {
		t.Errorf("expected the link to be active again")
	}
	if err := db.First(&stored, link.ID).Error; err != nil {
		t.Fatalf("failed to reload the link: %v", err)
	}
	if !stored.IsActive || stored.Title != "YouTube (paused)" {
		t.Errorf("expected the link active with its title preserved, got %+v", stored)
	}

	other := performJSON(t, router, http.MethodPost, "/api/v1/admin/links", map[string]any{
		"slug":        "other",
		"title":       "Other",
		"target_type": "landing",
	})
	expectStatus(t, other, http.StatusCreated)
	otherLink := decodeJSON[models.CampaignLink](t, other)

	collision := performJSON(t, router, http.MethodPut, "/api/v1/admin/links/"+strconv.FormatUint(uint64(otherLink.ID), 10), map[string]any{
		"slug": "ytb",
	})
	expectStatus(t, collision, http.StatusConflict)

	invalid := performJSON(t, router, http.MethodPut, "/api/v1/admin/links/"+strconv.FormatUint(uint64(link.ID), 10), map[string]any{
		"target_type": "channel",
	})
	expectStatus(t, invalid, http.StatusBadRequest)

	invalidSlug := performJSON(t, router, http.MethodPut, "/api/v1/admin/links/"+strconv.FormatUint(uint64(link.ID), 10), map[string]any{
		"slug": "!!!",
	})
	expectStatus(t, invalidSlug, http.StatusBadRequest)

	missing := performJSON(t, router, http.MethodPut, "/api/v1/admin/links/999999", map[string]any{"title": "nope"})
	expectStatus(t, missing, http.StatusNotFound)

	badID := performJSON(t, router, http.MethodPut, "/api/v1/admin/links/abc", map[string]any{"title": "nope"})
	expectStatus(t, badID, http.StatusBadRequest)
}

func TestCampaignLinkAdmin_DeleteRemovesLinkAndClicks(t *testing.T) {
	db := linksAdminPGDB(t)
	router := newCampaignLinkAdminRouter(db)

	rec := performJSON(t, router, http.MethodPost, "/api/v1/admin/links", map[string]any{
		"slug":        "ytb",
		"title":       "YouTube",
		"target_type": "landing",
	})
	expectStatus(t, rec, http.StatusCreated)
	link := decodeJSON[models.CampaignLink](t, rec)
	if err := db.Create(&models.CampaignLinkClick{LinkID: link.ID, VisitorID: "vabcdef123456"}).Error; err != nil {
		t.Fatalf("failed to seed a click: %v", err)
	}

	deleted := performJSON(t, router, http.MethodDelete, "/api/v1/admin/links/"+strconv.FormatUint(uint64(link.ID), 10), nil)
	expectStatus(t, deleted, http.StatusOK)

	var links, clicks int64
	if err := db.Model(&models.CampaignLink{}).Count(&links).Error; err != nil {
		t.Fatalf("failed to count links: %v", err)
	}
	if err := db.Model(&models.CampaignLinkClick{}).Count(&clicks).Error; err != nil {
		t.Fatalf("failed to count clicks: %v", err)
	}
	if links != 0 || clicks != 0 {
		t.Fatalf("expected the link and its clicks to be gone, got %d links and %d clicks", links, clicks)
	}

	again := performJSON(t, router, http.MethodDelete, "/api/v1/admin/links/"+strconv.FormatUint(uint64(link.ID), 10), nil)
	expectStatus(t, again, http.StatusNotFound)
}

func TestCampaignLinkAdmin_ListIncludesClickCounts(t *testing.T) {
	db := linksAdminPGDB(t)
	router := newCampaignLinkAdminRouter(db)

	busy := models.CampaignLink{Slug: "busy", Title: "Busy link", TargetType: "landing", IsActive: true}
	quiet := models.CampaignLink{Slug: "quiet", Title: "Quiet link", TargetType: "landing", IsActive: true}
	seedCampaignLink(t, db, &busy)
	seedCampaignLink(t, db, &quiet)
	if err := db.Create(&models.CampaignLinkClick{LinkID: busy.ID, VisitorID: "vaaaaaaaaaaaaaaaa"}).Error; err != nil {
		t.Fatalf("failed to seed a click: %v", err)
	}
	if err := db.Create(&models.CampaignLinkClick{LinkID: busy.ID, VisitorID: "vaaaaaaaaaaaaaaaa"}).Error; err != nil {
		t.Fatalf("failed to seed a click: %v", err)
	}
	if err := db.Create(&models.CampaignLinkClick{LinkID: busy.ID, VisitorID: "vbbbbbbbbbbbbbbbb"}).Error; err != nil {
		t.Fatalf("failed to seed a click: %v", err)
	}
	// A click without a visitor id is still a click, but never a visitor.
	if err := db.Create(&models.CampaignLinkClick{LinkID: busy.ID, VisitorID: ""}).Error; err != nil {
		t.Fatalf("failed to seed a click: %v", err)
	}

	rec := performJSON(t, router, http.MethodGet, "/api/v1/admin/links", nil)
	expectStatus(t, rec, http.StatusOK)

	links := decodeJSON[[]CampaignLinkWithClicks](t, rec)
	if len(links) != 2 {
		t.Fatalf("expected two links, got %d", len(links))
	}
	bySlug := make(map[string]CampaignLinkWithClicks, len(links))
	for _, link := range links {
		bySlug[link.Slug] = link
	}
	if got := bySlug["busy"]; got.Clicks != 4 || got.UniqueVisitors != 2 {
		t.Errorf("expected busy to have 4 clicks / 2 visitors, got %d / %d", got.Clicks, got.UniqueVisitors)
	}
	if got := bySlug["quiet"]; got.Clicks != 0 || got.UniqueVisitors != 0 {
		t.Errorf("expected quiet to have no clicks, got %d / %d", got.Clicks, got.UniqueVisitors)
	}
}

func TestCampaignLinkReport_AggregatesTrafficAndMoney(t *testing.T) {
	db := linksAdminPGDB(t)
	router := newCampaignLinkAdminRouter(db)
	now := time.Now().UTC().Truncate(time.Second)

	link := models.CampaignLink{Slug: "ytb", Title: "YouTube", TargetType: "landing", IsActive: true}
	quiet := models.CampaignLink{Slug: "quiet", Title: "Quiet", TargetType: "channel", TargetValue: "https://t.me/spark_prep_channel", IsActive: true}
	// A paused link keeps its history but stops collecting clicks; the report
	// still lists it (zeroed here) so past periods never lose a row.
	paused := models.CampaignLink{Slug: "paused", Title: "Paused", TargetType: "landing", IsActive: true}
	seedCampaignLink(t, db, &link)
	seedCampaignLink(t, db, &quiet)
	seedCampaignLink(t, db, &paused)
	if err := db.Model(&models.CampaignLink{}).Where("id = ?", paused.ID).Update("is_active", false).Error; err != nil {
		t.Fatalf("failed to pause the link: %v", err)
	}

	const (
		visitorOne = "vaaaaaaaaaaaaaaaa"
		visitorTwo = "vbbbbbbbbbbbbbbbb"
	)
	for _, click := range []models.CampaignLinkClick{
		{LinkID: link.ID, VisitorID: visitorOne, CreatedAt: now},
		{LinkID: link.ID, VisitorID: visitorOne, CreatedAt: now},
		{LinkID: link.ID, VisitorID: visitorTwo, CreatedAt: now},
	} {
		if err := db.Create(&click).Error; err != nil {
			t.Fatalf("failed to seed a click: %v", err)
		}
	}
	for _, event := range []models.AnalyticsEvent{
		{VisitorID: visitorOne, Step: "landing_view", CreatedAt: now},
		{VisitorID: visitorOne, Step: "bot_lead", CreatedAt: now},
	} {
		if err := db.Create(&event).Error; err != nil {
			t.Fatalf("failed to seed an event: %v", err)
		}
	}
	if err := db.Create(&models.PaymentTransaction{
		PaymentID: "pay_1",
		VisitorID: visitorOne,
		Amount:    10900,
		Status:    "SUCCESS",
		Currency:  "RUB",
		CreatedAt: now,
	}).Error; err != nil {
		t.Fatalf("failed to seed a payment: %v", err)
	}
	// Money without a visitor id is reported globally, not per link.
	if err := db.Create(&models.PaymentTransaction{
		PaymentID: "pay_orphan",
		Amount:    5000,
		Status:    "SUCCESS",
		Currency:  "RUB",
		CreatedAt: now,
	}).Error; err != nil {
		t.Fatalf("failed to seed an unattributed payment: %v", err)
	}

	query := url.Values{}
	query.Set("from", now.Add(-time.Hour).Format(time.RFC3339))
	query.Set("to", now.Add(time.Hour).Format(time.RFC3339))
	rec := performJSON(t, router, http.MethodGet, "/api/v1/admin/links/report?"+query.Encode(), nil)
	expectStatus(t, rec, http.StatusOK)

	report := decodeJSON[LinkReportResponse](t, rec)
	if len(report.Rows) != 3 {
		t.Fatalf("expected three report rows, got %d (%+v)", len(report.Rows), report.Rows)
	}
	bySlug := make(map[string]LinkReportRow, len(report.Rows))
	for _, candidate := range report.Rows {
		bySlug[candidate.Slug] = candidate
	}
	row := report.Rows[0]
	if row.Slug != "ytb" {
		t.Fatalf("expected the busiest link first, got %q", row.Slug)
	}
	if row.Clicks != 3 {
		t.Errorf("expected 3 clicks, got %d", row.Clicks)
	}
	if row.UniqueVisitors != 2 {
		t.Errorf("expected 2 unique visitors, got %d", row.UniqueVisitors)
	}
	if row.SiteVisits != 1 {
		t.Errorf("expected 1 site visit, got %d", row.SiteVisits)
	}
	if row.Leads != 1 {
		t.Errorf("expected 1 lead, got %d", row.Leads)
	}
	if row.Payments != 1 {
		t.Errorf("expected 1 payment, got %d", row.Payments)
	}
	assertFloat(t, "revenue", row.Revenue, 10900)
	assertFloat(t, "epc", row.EPC, 3633.33)
	assertFloat(t, "rpv", row.RPV, 5450)

	for _, slug := range []string{"quiet", "paused"} {
		empty, ok := bySlug[slug]
		if !ok {
			t.Fatalf("expected the %q link in the report", slug)
		}
		if empty.Clicks != 0 || empty.UniqueVisitors != 0 || empty.SiteVisits != 0 || empty.Leads != 0 || empty.Payments != 0 {
			t.Errorf("expected a zeroed row for %q, got %+v", slug, empty)
		}
		assertFloat(t, slug+" revenue", empty.Revenue, 0)
		assertFloat(t, slug+" epc", empty.EPC, 0)
		assertFloat(t, slug+" rpv", empty.RPV, 0)
	}

	assertFloat(t, "unattributed revenue", report.UnattributedRevenue, 5000)
}

func TestCampaignLinkReport_WindowFiltersEverySource(t *testing.T) {
	db := linksAdminPGDB(t)
	router := newCampaignLinkAdminRouter(db)
	now := time.Now().UTC().Truncate(time.Second)
	old := now.Add(-48 * time.Hour)

	link := models.CampaignLink{Slug: "ytb", Title: "YouTube", TargetType: "landing", IsActive: true}
	seedCampaignLink(t, db, &link)

	const visitor = "vcccccccccccccccc"
	for _, click := range []models.CampaignLinkClick{
		{LinkID: link.ID, VisitorID: visitor, CreatedAt: old},
		{LinkID: link.ID, VisitorID: visitor, CreatedAt: now},
	} {
		if err := db.Create(&click).Error; err != nil {
			t.Fatalf("failed to seed a click: %v", err)
		}
	}
	if err := db.Create(&models.AnalyticsEvent{VisitorID: visitor, Step: "landing_view", CreatedAt: old}).Error; err != nil {
		t.Fatalf("failed to seed an old event: %v", err)
	}
	if err := db.Create(&models.PaymentTransaction{
		PaymentID: "pay_old",
		VisitorID: visitor,
		Amount:    10900,
		Status:    "SUCCESS",
		CreatedAt: old,
	}).Error; err != nil {
		t.Fatalf("failed to seed an old payment: %v", err)
	}

	query := url.Values{}
	query.Set("from", now.Add(-time.Hour).Format(time.RFC3339))
	query.Set("to", now.Add(time.Hour).Format(time.RFC3339))
	rec := performJSON(t, router, http.MethodGet, "/api/v1/admin/links/report?"+query.Encode(), nil)
	expectStatus(t, rec, http.StatusOK)

	report := decodeJSON[LinkReportResponse](t, rec)
	if len(report.Rows) != 1 {
		t.Fatalf("expected one report row, got %d", len(report.Rows))
	}
	row := report.Rows[0]
	if row.Clicks != 1 || row.UniqueVisitors != 1 {
		t.Errorf("expected only the in-window click, got %d clicks / %d visitors", row.Clicks, row.UniqueVisitors)
	}
	if row.SiteVisits != 0 {
		t.Errorf("expected the old landing view to be excluded, got %d", row.SiteVisits)
	}
	if row.Payments != 0 {
		t.Errorf("expected the old payment to be excluded, got %d", row.Payments)
	}
	assertFloat(t, "revenue", row.Revenue, 0)
	assertFloat(t, "unattributed revenue", report.UnattributedRevenue, 0)
}

func TestCampaignLinkReport_RejectsBadWindow(t *testing.T) {
	db := linksAdminPGDB(t)
	router := newCampaignLinkAdminRouter(db)

	cases := []string{
		"from=not-a-date",
		"to=2026-13-45",
		"from=2026-10-20&to=2026-10-19",
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			rec := performJSON(t, router, http.MethodGet, "/api/v1/admin/links/report?"+raw, nil)
			expectStatus(t, rec, http.StatusBadRequest)
		})
	}
}

func TestCampaignLinkAdmin_CreateInactivePersistsFalse(t *testing.T) {
	db := linksAdminPGDB(t)
	router := newCampaignLinkAdminRouter(db)

	rec := performJSON(t, router, http.MethodPost, "/api/v1/admin/links", map[string]any{
		"slug":        "draft-link",
		"title":       "Draft",
		"target_type": "landing",
		"is_active":   false,
	})
	expectStatus(t, rec, http.StatusCreated)
	created := decodeJSON[models.CampaignLink](t, rec)
	if created.IsActive {
		t.Errorf("expected the response to report an inactive link")
	}

	var stored models.CampaignLink
	if err := db.First(&stored, created.ID).Error; err != nil {
		t.Fatalf("failed to reload the link: %v", err)
	}
	if stored.IsActive {
		t.Fatalf("expected an explicitly inactive link to stay inactive in the database")
	}
}

func TestCampaignLinkReport_AttributesEachPaymentToOneLink(t *testing.T) {
	db := linksAdminPGDB(t)
	router := newCampaignLinkAdminRouter(db)
	now := time.Now().UTC().Truncate(time.Second)

	first := models.CampaignLink{Slug: "link-a", Title: "First", TargetType: "landing", IsActive: true}
	second := models.CampaignLink{Slug: "link-b", Title: "Second", TargetType: "landing", IsActive: true}
	seedCampaignLink(t, db, &first)
	seedCampaignLink(t, db, &second)

	const visitor = "vdddddddddddddddd"
	for _, click := range []models.CampaignLinkClick{
		{LinkID: first.ID, VisitorID: visitor, CreatedAt: now.Add(-2 * time.Hour)},
		{LinkID: second.ID, VisitorID: visitor, CreatedAt: now.Add(-time.Hour)},
	} {
		if err := db.Create(&click).Error; err != nil {
			t.Fatalf("failed to seed a click: %v", err)
		}
	}
	if err := db.Create(&models.PaymentTransaction{
		PaymentID: "pay_one",
		VisitorID: visitor,
		Amount:    1000,
		Status:    "SUCCESS",
		CreatedAt: now,
	}).Error; err != nil {
		t.Fatalf("failed to seed a payment: %v", err)
	}

	query := url.Values{}
	query.Set("from", now.Add(-3*time.Hour).Format(time.RFC3339))
	query.Set("to", now.Add(time.Hour).Format(time.RFC3339))
	rec := performJSON(t, router, http.MethodGet, "/api/v1/admin/links/report?"+query.Encode(), nil)
	expectStatus(t, rec, http.StatusOK)

	report := decodeJSON[LinkReportResponse](t, rec)
	bySlug := make(map[string]LinkReportRow, len(report.Rows))
	for _, row := range report.Rows {
		bySlug[row.Slug] = row
	}
	if got := bySlug["link-a"]; got.Payments != 1 || math.Abs(got.Revenue-1000) > 0.005 {
		t.Fatalf("expected link-a (earliest click) to own the payment, got payments=%d revenue=%.2f", got.Payments, got.Revenue)
	}
	if got := bySlug["link-b"]; got.Payments != 0 || got.Revenue != 0 {
		t.Fatalf("expected link-b to have no money, got payments=%d revenue=%.2f", got.Payments, got.Revenue)
	}

	var rowsRevenue float64
	for _, row := range report.Rows {
		rowsRevenue += row.Revenue
	}
	assertFloat(t, "sum(rows.revenue) + unattributed", rowsRevenue+report.UnattributedRevenue, 1000)
}

func TestCampaignLinkReport_IgnoresClicksAfterPayment(t *testing.T) {
	db := linksAdminPGDB(t)
	router := newCampaignLinkAdminRouter(db)
	now := time.Now().UTC().Truncate(time.Second)

	link := models.CampaignLink{Slug: "late-click", Title: "Late click", TargetType: "landing", IsActive: true}
	seedCampaignLink(t, db, &link)

	const visitor = "veeeeeeeeeeeeeeee"
	if err := db.Create(&models.PaymentTransaction{
		PaymentID: "pay_before_click",
		VisitorID: visitor,
		Amount:    1000,
		Status:    "SUCCESS",
		CreatedAt: now,
	}).Error; err != nil {
		t.Fatalf("failed to seed a payment: %v", err)
	}
	if err := db.Create(&models.CampaignLinkClick{
		LinkID:    link.ID,
		VisitorID: visitor,
		CreatedAt: now.Add(20 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("failed to seed a click: %v", err)
	}

	query := url.Values{}
	query.Set("from", now.Add(-time.Hour).Format(time.RFC3339))
	query.Set("to", now.Add(time.Hour).Format(time.RFC3339))
	rec := performJSON(t, router, http.MethodGet, "/api/v1/admin/links/report?"+query.Encode(), nil)
	expectStatus(t, rec, http.StatusOK)

	report := decodeJSON[LinkReportResponse](t, rec)
	if len(report.Rows) != 1 {
		t.Fatalf("expected one report row, got %d", len(report.Rows))
	}
	if got := report.Rows[0]; got.Payments != 0 || got.Revenue != 0 {
		t.Fatalf("a click after the payment must not own it, got payments=%d revenue=%.2f", got.Payments, got.Revenue)
	}
	assertFloat(t, "unattributed revenue", report.UnattributedRevenue, 1000)
}

func TestCampaignLinkReport_ExcludesFailedAndPendingPayments(t *testing.T) {
	db := linksAdminPGDB(t)
	router := newCampaignLinkAdminRouter(db)
	now := time.Now().UTC().Truncate(time.Second)

	link := models.CampaignLink{Slug: "no-money", Title: "No money", TargetType: "landing", IsActive: true}
	seedCampaignLink(t, db, &link)

	const visitor = "vffffffffffffffff"
	if err := db.Create(&models.CampaignLinkClick{
		LinkID:    link.ID,
		VisitorID: visitor,
		CreatedAt: now.Add(-time.Hour),
	}).Error; err != nil {
		t.Fatalf("failed to seed a click: %v", err)
	}
	for _, payment := range []models.PaymentTransaction{
		{PaymentID: "pay_fail", VisitorID: visitor, Amount: 1000, Status: "FAIL", CreatedAt: now},
		{PaymentID: "pay_pending", VisitorID: visitor, Amount: 500, Status: "PENDING", CreatedAt: now},
	} {
		if err := db.Create(&payment).Error; err != nil {
			t.Fatalf("failed to seed a payment: %v", err)
		}
	}

	query := url.Values{}
	query.Set("from", now.Add(-2*time.Hour).Format(time.RFC3339))
	query.Set("to", now.Add(time.Hour).Format(time.RFC3339))
	rec := performJSON(t, router, http.MethodGet, "/api/v1/admin/links/report?"+query.Encode(), nil)
	expectStatus(t, rec, http.StatusOK)

	report := decodeJSON[LinkReportResponse](t, rec)
	if len(report.Rows) != 1 {
		t.Fatalf("expected one report row, got %d", len(report.Rows))
	}
	if got := report.Rows[0]; got.Payments != 0 || got.Revenue != 0 {
		t.Fatalf("only SUCCESS payments count, got payments=%d revenue=%.2f", got.Payments, got.Revenue)
	}
	assertFloat(t, "unattributed revenue", report.UnattributedRevenue, 0)
}

func TestCampaignLinkReport_DoesNotCountEmptyVisitorAsUnique(t *testing.T) {
	db := linksAdminPGDB(t)
	router := newCampaignLinkAdminRouter(db)
	now := time.Now().UTC().Truncate(time.Second)

	link := models.CampaignLink{Slug: "anonymous", Title: "Anonymous", TargetType: "landing", IsActive: true}
	seedCampaignLink(t, db, &link)
	for _, click := range []models.CampaignLinkClick{
		{LinkID: link.ID, VisitorID: "", CreatedAt: now},
		{LinkID: link.ID, VisitorID: "vaaaaaaaaaaaaaaab", CreatedAt: now},
	} {
		if err := db.Create(&click).Error; err != nil {
			t.Fatalf("failed to seed a click: %v", err)
		}
	}

	query := url.Values{}
	query.Set("from", now.Add(-time.Hour).Format(time.RFC3339))
	query.Set("to", now.Add(time.Hour).Format(time.RFC3339))
	rec := performJSON(t, router, http.MethodGet, "/api/v1/admin/links/report?"+query.Encode(), nil)
	expectStatus(t, rec, http.StatusOK)

	report := decodeJSON[LinkReportResponse](t, rec)
	if len(report.Rows) != 1 {
		t.Fatalf("expected one report row, got %d", len(report.Rows))
	}
	if got := report.Rows[0]; got.Clicks != 2 || got.UniqueVisitors != 1 {
		t.Fatalf("expected 2 clicks / 1 unique visitor, got %d / %d", got.Clicks, got.UniqueVisitors)
	}
}

func TestCampaignLinkReport_DateOnlyToIncludesToday(t *testing.T) {
	db := linksAdminPGDB(t)
	router := newCampaignLinkAdminRouter(db)
	now := time.Now()

	link := models.CampaignLink{Slug: "today", Title: "Today", TargetType: "landing", IsActive: true}
	seedCampaignLink(t, db, &link)
	if err := db.Create(&models.CampaignLinkClick{
		LinkID:    link.ID,
		VisitorID: "vaaaaaaaaaaaaaaac",
		CreatedAt: now,
	}).Error; err != nil {
		t.Fatalf("failed to seed a click: %v", err)
	}

	today := now.In(campaignReportZone).Format(campaignReportDateLayout)
	rec := performJSON(t, router, http.MethodGet, "/api/v1/admin/links/report?to="+today, nil)
	expectStatus(t, rec, http.StatusOK)

	report := decodeJSON[LinkReportResponse](t, rec)
	if len(report.Rows) != 1 {
		t.Fatalf("expected one report row, got %d", len(report.Rows))
	}
	if got := report.Rows[0]; got.Clicks != 1 || got.UniqueVisitors != 1 {
		t.Fatalf("a date-only to must cover the whole current day, got clicks=%d visitors=%d", got.Clicks, got.UniqueVisitors)
	}
}
