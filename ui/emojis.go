package ui

// Status emoji constants for consistent UI display
const (
	// Pass/Fail indicators (light weight, matching style)
	EmojiCheckMark = "✓" // U+2713 - Check mark (approved/pass)
	EmojiBallotX   = "✗" // U+2717 - Ballot X (rejected/fail, matches check mark style)

	// Heavy weight pass/fail (more prominent)
	EmojiPassesPolicy   = "✅" // U+2705 - White heavy check mark (success/approved)
	EmojiViolatesPolicy = "❌" // U+274C - Cross mark (error/rejected)

	EmojiApproved = "👍" // U+1F44D - Thumbs up (approved/positive
	EmojiRejected = "👎" // U+1F44E - Thumbs down (rejected/negative)

	// Status indicators
	EmojiNew      = "🆕" // U+1F195 - New
	EmojiOpen     = "🔴" // U+1F534 - Red circle (open/active)
	EmojiReopened = "🔄" // U+1F504 - Counterclockwise arrows (reopened)
	EmojiPending  = "⏳" // U+23F3 - Hourglass (pending/in progress)
	EmojiUnknown  = "❓" // U+2753 - Question mark (unknown status)
	EmojiComment  = "💬" // U+1F4AC - Speech balloon (comment)
)
