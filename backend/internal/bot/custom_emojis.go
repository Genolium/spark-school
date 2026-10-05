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

	// Additional Verified Custom Emojis from latest pack:
	EmojiStarBigGreen = "5886278646540280100" // 🤩 Green big sparkle
	EmojiSpeechGreen  = "5884113368842772574" // 🗣 Speech / Talk
	EmojiWarning      = "5409346180704388243" // ⚠️ Warning / Alert
	EmojiPromoSparkle = "5386789158494680328" // ✨ Magic sparkle for promo
	EmojiHeartGreen   = "5884256009001639539" // 💚 Green heart
	EmojiSpiralGreen  = "5884282156762537884" // 🌀 Green spiral
	EmojiSunGreen     = "5883980366590517942" // ☀️ Green sun/flower
	EmojiArrowDown    = "5884233340164251596" // ⬇️ Green arrow down
	EmojiChatBubbles  = "5884512740671757591" // 💬 Green chat bubbles
	EmojiLightning    = "5348243699618829694" // ⚡️ Lightning bolt
	EmojiPaperclip    = "5348212037119924779" // 📎 Paperclip
	EmojiCalendarTag  = "5348508089215639697" // 🗓 Calendar / Tag

	// New detected emojis from user screenshots:
	EmojiCheckCircleGreen = "5884152869656994006" // ✅ Green check circle
	EmojiCheckVGreen      = "5884052947242852173" // ✔️ Green checkmark V
	EmojiPaperclipGreen   = "5884282156762537884" // 📎 Green paperclip/spiral
	EmojiHeartSolidGreen  = "5884421867753709341" // 💚 Solid green heart
	EmojiArrowRightGreen  = "5884424573583106260" // ➔ Green arrow right

	// Aliases for backward compatibility
	EmojiSparkle1 = EmojiSparkleGreen
	EmojiSparkle2 = EmojiStarGreen
	EmojiCheck    = EmojiSunGreen
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
