package bot

import (
	"fmt"
	"regexp"
)

// Verified Custom Emoji IDs from user's live packs
const (
	// Green / Spark theme pack
	EmojiSparkleGreen = "5884159398007283867" // Sparkle ✨
	EmojiStarGreen    = "5884407720131436294" // Star ⭐

	// User's custom pack (from live /emoji diagnostic)
	EmojiTicket      = "5267231601279912008" // 🎟 Ticket / Promo
	EmojiPointDown   = "5470177992950946662" // 👇 Finger down
	EmojiBooks       = "5884435603059124160" // 📚 Books / Study materials
	EmojiSparkleBlue = "5348261296099844892" // ✨ Sparkle
	EmojiStarGold    = "5399919638022732210" // ⭐️ Star
	EmojiCard        = "5380098557225222946" // 💳 Credit card / Payment
	EmojiNum1        = "5348266445765632009" // 1️⃣ Number 1
	EmojiNum2        = "5348034027905375724" // 2️⃣ Number 2
	EmojiPinRed      = "5386376601116098709" // 📍 Location pin
	EmojiPinPush     = "5386429751336383258" // 📌 Pushpin
	EmojiIdea        = "5386435274664326894" // 💡 Lightbulb
	EmojiFire        = "5420315771991497307" // 🔥 Fire / Urgency

	// Aliases for backward compatibility
	EmojiSparkle1 = EmojiSparkleGreen
	EmojiSparkle2 = EmojiStarGreen
	EmojiCheck    = EmojiSparkleGreen
)

var tgEmojiRegex = regexp.MustCompile(`<tg-emoji[^>]*>(.*?)</tg-emoji>`)

// TgEmoji formats an HTML custom emoji tag with a standard Unicode fallback for non-premium clients.
func TgEmoji(id string, fallback string) string {
	return fmt.Sprintf(`<tg-emoji emoji-id="%s">%s</tg-emoji>`, id, fallback)
}

// StripTgEmoji removes <tg-emoji> tags and preserves only the fallback characters.
func StripTgEmoji(htmlText string) string {
	return tgEmojiRegex.ReplaceAllString(htmlText, "$1")
}
