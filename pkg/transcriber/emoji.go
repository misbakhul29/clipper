package transcriber

import (
	"regexp"
	"strings"
)

type emojiDefinition struct {
	emoji    string
	keywords []string
}

type compiledEmojiRule struct {
	emoji   string
	pattern *regexp.Regexp
}

var rawEmojiDefinitions = []emojiDefinition{
	{
		emoji: "💰",
		keywords: []string{
			"uang", "duit", "cuan", "dollar", "dolar", "rupiah", "kaya", "gaji",
			"omset", "profit", "untung", "harga", "mahal", "biaya", "modal", "jutaan",
			"miliaran", "triliun", "money", "rich", "cash", "millionaire", "billionaire",
		},
	},
	{
		emoji: "🔥",
		keywords: []string{
			"api", "panas", "membakar", "terbakar", "viral", "trending", "booming",
			"meledak", "semangat", "heboh", "fire", "hot", "burn", "flame",
		},
	},
	{
		emoji: "💡",
		keywords: []string{
			"ide", "pikir", "pikiran", "otak", "cerdas", "pintar", "solusi",
			"solusinya", "trik", "tips", "rahasia", "idea", "think", "brain",
			"smart", "clever", "solution",
		},
	},
	{
		emoji: "⚠️",
		keywords: []string{
			"bahaya", "peringatan", "awas", "waspada", "darurat", "ancaman", "hati-hati",
			"danger", "warning", "caution", "alert", "threat", "careful",
		},
	},
	{
		emoji: "💀",
		keywords: []string{
			"mati", "kematian", "tewas", "terbunuh", "membunuh", "korban", "kubur",
			"racun", "dead", "death", "die", "kill", "poison", "fatal",
		},
	},
	{
		emoji: "🧟",
		keywords: []string{
			"zombi", "zombie", "monster", "infeksi", "terinfeksi", "mayat", "hantu",
		},
	},
	{
		emoji: "🎯",
		keywords: []string{
			"target", "tujuan", "fokus", "arah", "sasaran", "goal", "goals",
			"focus", "aim", "achieve",
		},
	},
	{
		emoji: "🏆",
		keywords: []string{
			"menang", "juara", "sukses", "berhasil", "kejayaan", "pemenang",
			"prestasi", "win", "winner", "trophy", "champion", "success",
		},
	},
	{
		emoji: "❤️",
		keywords: []string{
			"cinta", "sayang", "kasih", "hati", "kekasih", "pacar", "jodoh",
			"bahagia", "love", "lover", "heart", "beloved",
		},
	},
	{
		emoji: "⏰",
		keywords: []string{
			"waktu", "jam", "menit", "detik", "terlambat", "segera", "cepat",
			"buru-buru", "time", "clock", "hour", "minute", "fast", "quick", "deadline",
		},
	},
	{
		emoji: "⚡",
		keywords: []string{
			"listrik", "petir", "kilat", "tenaga", "energi", "kekuatan", "dahsyat",
			"lightning", "electric", "power", "energy",
		},
	},
	{
		emoji: "🚀",
		keywords: []string{
			"roket", "terbang", "meroket", "peluncuran", "meluncur", "angkasa",
			"rocket", "launch", "skyrocket", "fly",
		},
	},
	{
		emoji: "🤯",
		keywords: []string{
			"kaget", "terkejut", "syok", "gila", "gokil", "parah", "mustahil",
			"shock", "shocked", "crazy", "insane", "impossible", "mindblown",
		},
	},
	{
		emoji: "❓",
		keywords: []string{
			"kenapa", "mengapa", "bagaimana", "siapa", "bingung", "misteri",
			"tanya", "pertanyaan", "why", "how", "what", "who", "confused", "mystery",
		},
	},
	{
		emoji: "📈",
		keywords: []string{
			"saham", "investasi", "crypto", "kripto", "trading", "naik", "tumbuh",
			"pertumbuhan", "meningkat", "stock", "stocks", "growth", "invest", "chart",
		},
	},
	{
		emoji: "🤖",
		keywords: []string{
			"robot", "ai", "komputer", "teknologi", "sistem", "coding", "program",
			"software", "aplikasi", "tech", "computer", "technology", "algorithm",
		},
	},
	{
		emoji: "🔑",
		keywords: []string{
			"kunci", "rahasianya", "bocoran", "terbuka", "secret", "key", "unlock",
		},
	},
	{
		emoji: "🍔",
		keywords: []string{
			"makan", "makanan", "kuliner", "lapar", "lezat", "enak", "restoran",
			"food", "eat", "delicious", "hungry", "snack",
		},
	},
	{
		emoji: "😴",
		keywords: []string{
			"tidur", "lelah", "capek", "istirahat", "ngantuk", "sleep", "tired", "rest",
		},
	},
	{
		emoji: "⭐",
		keywords: []string{
			"bintang", "terbaik", "hebat", "keren", "mantap", "istimewa", "top",
			"star", "best", "awesome", "super",
		},
	},
}

var compiledEmojiRules []compiledEmojiRule

func init() {
	for _, def := range rawEmojiDefinitions {
		// Build regex pattern matching whole words: (?i)\b(kw1|kw2|...)\b
		patternStr := `(?i)\b(` + strings.Join(def.keywords, "|") + `)\b`
		re := regexp.MustCompile(patternStr)
		compiledEmojiRules = append(compiledEmojiRules, compiledEmojiRule{
			emoji:   def.emoji,
			pattern: re,
		})
	}
}

// ContainsEmoji checks whether s already contains standard Unicode emoji characters.
func ContainsEmoji(s string) bool {
	for _, r := range s {
		if (r >= 0x1F600 && r <= 0x1F64F) || // Emoticons
			(r >= 0x1F300 && r <= 0x1F5FF) || // Misc Symbols and Pictographs
			(r >= 0x1F680 && r <= 0x1F6FF) || // Transport and Map
			(r >= 0x1F900 && r <= 0x1F9FF) || // Supplemental Symbols
			(r >= 0x1FA00 && r <= 0x1FAFF) || // Symbols and Pictographs Extended-A
			(r >= 0x2600 && r <= 0x26FF) || // Misc Symbols (⚠️, ⚡, etc.)
			(r >= 0x2700 && r <= 0x27BF) { // Dingbats (❤️, ⭐, ❓, etc.)
			return true
		}
	}
	return false
}

// FindContextualEmoji inspects text for semantic keywords and returns the highest-priority matching emoji.
func FindContextualEmoji(text string) string {
	for _, rule := range compiledEmojiRules {
		if rule.pattern.MatchString(text) {
			return rule.emoji
		}
	}
	return ""
}

// InjectContextualEmojis appends relevant emojis to subtitle entries with pacing and consecutive debounce.
func InjectContextualEmojis(entries []SubtitleEntry) []SubtitleEntry {
	var lastEmoji string
	var result []SubtitleEntry

	for _, entry := range entries {
		// If entry already contains an emoji, don't inject another
		if ContainsEmoji(entry.Text) {
			lastEmoji = ""
			result = append(result, entry)
			continue
		}

		emoji := FindContextualEmoji(entry.Text)
		if emoji != "" && emoji != lastEmoji {
			entry.Text = entry.Text + " " + emoji
			lastEmoji = emoji
		} else {
			lastEmoji = ""
		}

		result = append(result, entry)
	}

	return result
}
