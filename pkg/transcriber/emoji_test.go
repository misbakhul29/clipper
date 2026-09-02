package transcriber

import (
	"testing"
)

func TestFindContextualEmoji(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"banyak uang masuk", "💰"},
		{"punya cuan melimpah", "💰"},
		{"terbakar api asmara", "🔥"},
		{"ini konten viral banget", "🔥"},
		{"saya punya ide brilian", "💡"},
		{"ada bahaya mengintai", "⚠️"},
		{"dia sudah mati", "💀"},
		{"serangan zombi ganas", "🧟"},
		{"fokus pada target kita", "🎯"},
		{"kita pasti menang", "🏆"},
		{"jatuh cinta padanya", "❤️"},
		{"waktu sudah habis", "⏰"},
		{"tenaga listrik dahsyat", "⚡"},
		{"terbang naik roket", "🚀"},
		{"semua orang kaget", "🤯"},
		{"mengapa bisa terjadi", "❓"},
		{"investasi crypto naik", "📈"},
		{"sistem robot otomatis", "🤖"},
		{"ini rahasia sukses", "💡"}, // Note: ide/rahasia matches 💡
		{"bocoran kunci utama", "🔑"},
		{"makanan sangat lezat", "🍔"},
		{"badan capek mau tidur", "😴"},
		{"kamu memang terbaik", "⭐"},
		// Negative check: word boundary should NOT match substring "rapi" for "api"
		{"bajunya sangat rapi sekali", ""},
		// English checks
		{"make a lot of money", "💰"},
		{"this is so hot and trending", "🔥"},
		{"great smart solution", "💡"},
	}

	for _, c := range cases {
		got := FindContextualEmoji(c.input)
		if got != c.expected {
			t.Errorf("FindContextualEmoji(%q) = %q, want %q", c.input, got, c.expected)
		}
	}
}

func TestContainsEmoji(t *testing.T) {
	if !ContainsEmoji("Halo 💰 dunia") {
		t.Errorf("expected true for string containing 💰")
	}
	if !ContainsEmoji("Api membara 🔥") {
		t.Errorf("expected true for string containing 🔥")
	}
	if !ContainsEmoji("Hati ❤️ ku") {
		t.Errorf("expected true for string containing ❤️")
	}
	if ContainsEmoji("Teks biasa tanpa emoji") {
		t.Errorf("expected false for string without emoji")
	}
}

func TestInjectContextualEmojis(t *testing.T) {
	entries := []SubtitleEntry{
		{Start: "0:00:01.00", End: "0:00:02.00", Text: "BANYAK UANG"},
		{Start: "0:00:02.00", End: "0:00:03.00", Text: "UANG MELIMPAH"}, // Consecutive duplicate trigger
		{Start: "0:00:03.00", End: "0:00:04.00", Text: "TERBAKAR API"},
		{Start: "0:00:04.00", End: "0:00:05.00", Text: "SUDAH ADA EMOJI 🚀 DI SINI"},
		{Start: "0:00:05.00", End: "0:00:06.00", Text: "KALIMAT BIASA SAJA"},
	}

	res := InjectContextualEmojis(entries)
	if len(res) != len(entries) {
		t.Fatalf("expected %d entries, got %d", len(entries), len(res))
	}

	// Entry 0 should have 💰
	if res[0].Text != "BANYAK UANG 💰" {
		t.Errorf("entry 0 mismatch: %q", res[0].Text)
	}
	// Entry 1 should NOT repeat 💰 consecutively
	if res[1].Text != "UANG MELIMPAH" {
		t.Errorf("entry 1 expected debounce without duplicate 💰, got: %q", res[1].Text)
	}
	// Entry 2 should have 🔥
	if res[2].Text != "TERBAKAR API 🔥" {
		t.Errorf("entry 2 mismatch: %q", res[2].Text)
	}
	// Entry 3 already has emoji 🚀, should remain untouched
	if res[3].Text != "SUDAH ADA EMOJI 🚀 DI SINI" {
		t.Errorf("entry 3 mismatch: %q", res[3].Text)
	}
	// Entry 4 no match, should remain untouched
	if res[4].Text != "KALIMAT BIASA SAJA" {
		t.Errorf("entry 4 mismatch: %q", res[4].Text)
	}
}
