package bot

import (
	"fmt"
)

// Verified Custom Emoji IDs from @emojiabc (abc2409_by_TgEmojiBot)
const (
	EmojiSparkle1 = "5884347277056679787" // Sparkle / Star ✦
	EmojiSparkle2 = "5884407720131436294" // Sparkle ✧
	EmojiHeart    = "5884256009001639539" // Heart
	EmojiCat      = "5884204662667617270" // Cat
	EmojiCamera   = "5884048695225229779" // Camera / Travel
	EmojiMoon     = "5884122117691155031" // Moon
	EmojiBooks    = "5884443560305912416" // Books / Study
	EmojiCoffee   = "5884217637763817845" // Coffee
	EmojiCup      = "5884021581096689340" // Cup
	EmojiSun      = "5884455295484174273" // Sun
	EmojiCheck    = "5884340336389532127" // Checkmark
)

// TgEmoji formats an HTML custom emoji tag with a fallback character for non-premium clients.
func TgEmoji(id string, fallback string) string {
	return fmt.Sprintf(`<tg-emoji emoji-id="%s">%s</tg-emoji>`, id, fallback)
}
