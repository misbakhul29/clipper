package transcriber

import (
	"testing"
)

func TestCleanSubtitleGarbage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Removes bracketed sounds and symbols",
			input:    "[Musik] Halo semuanya {tertawa} apa kabar? <c>mantap</c> >=+",
			expected: "Halo semuanya apa kabar? mantap",
		},
		{
			name:     "Removes narrator and speaker tags",
			input:    "Narrator: Selamat datang di channel ini!",
			expected: "Selamat datang di channel ini!",
		},
		{
			name:     "Removes parenthetical laughter and strange characters",
			input:    "(Laughter) Hari ini kita akan coding ><+= tutorial.",
			expected: "Hari ini kita akan coding tutorial.",
		},
		{
			name:     "Preserves normal dialogue text and numbers",
			input:    "Harga barang ini adalah $50 atau 750000 rupiah.",
			expected: "Harga barang ini adalah $50 atau 750000 rupiah.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := CleanSubtitleGarbage(tc.input)
			if result != tc.expected {
				t.Errorf("CleanSubtitleGarbage(%q) = %q; want %q", tc.input, result, tc.expected)
			}
		})
	}
}
