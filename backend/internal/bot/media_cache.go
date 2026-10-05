package bot

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

// MediaAsset keys
const (
	MediaMainBanner    = "bot-main.png"
	MediaGrantChances  = "bot-grant-chances.png"
	MediaBuyCourse     = "bot-buy-course.png"
)

// MediaCache holds resolved Telegram file IDs in memory and disk/db.
type MediaCache struct {
	mu      sync.RWMutex
	fileIDs map[string]string
	baseDir string
}

func NewMediaCache(baseDir string) *MediaCache {
	return &MediaCache{
		fileIDs: make(map[string]string),
		baseDir: baseDir,
	}
}

// Get returns the cached Telegram file ID if available.
func (mc *MediaCache) Get(key string) (string, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	id, ok := mc.fileIDs[key]
	return id, ok && id != ""
}

// Set stores the cached Telegram file ID.
func (mc *MediaCache) Set(key, fileID string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.fileIDs[key] = fileID
}

// ResolvePhotoFile returns telego.InputFile using cached file ID or local disk file.
func (s *BotService) ResolvePhotoFile(key string) telego.InputFile {
	if s.mediaCache != nil {
		if fileID, ok := s.mediaCache.Get(key); ok && fileID != "" {
			return tu.FileFromID(fileID)
		}
	}

	// Look up on disk
	localPath := s.findLocalMedia(key)
	if localPath != "" {
		if file, err := os.Open(localPath); err == nil {
			return tu.File(file)
		}
	}

	// Fallback to static verified constants if file not found
	switch key {
	case MediaMainBanner:
		return tu.FileFromID(PhotoGoldenGate)
	case MediaBuyCourse:
		return tu.FileFromID(PhotoProgramInfo)
	case MediaGrantChances:
		return tu.FileFromID(PhotoInterview)
	default:
		return tu.FileFromID(PhotoGoldenGate)
	}
}

// CacheSentPhoto extracts Telegram file ID from sent message and stores it for future instant sends.
func (s *BotService) CacheSentPhoto(key string, msg *telego.Message) {
	if s.mediaCache == nil || msg == nil || len(msg.Photo) == 0 {
		return
	}
	// Telegram sends multiple resolutions; highest is the last item
	highestRes := msg.Photo[len(msg.Photo)-1]
	if highestRes.FileID != "" {
		s.mediaCache.Set(key, highestRes.FileID)
		log.Printf("[Bot Media] Successfully cached Telegram file_id for %s: %s", key, highestRes.FileID)
	}
}

func (s *BotService) findLocalMedia(fileName string) string {
	candidates := []string{
		filepath.Join("/app", "media", fileName),
		filepath.Join("media", fileName),
		filepath.Join("backend", "media", fileName),
		filepath.Join("..", "backend", "media", fileName),
		filepath.Join("..", "media", fileName),
		filepath.Join("D:\\Documents\\Projects\\spark-school\\backend\\media", fileName),
	}

	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			log.Printf("[Bot Media] Found local media file: %s (%d bytes)", p, info.Size())
			return p
		}
	}
	log.Printf("[Bot Media] Local media file not found for: %s. Using permanent fallback file_id.", fileName)
	return ""
}
