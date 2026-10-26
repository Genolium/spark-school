package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"
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
