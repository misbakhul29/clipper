package transcriber

import (
	"testing"
)

func TestParseVTT(t *testing.T) {
	sampleVTT := `WEBVTT
Kind: captions
Language: en

00:00:01.000 --> 00:00:04.500
Hello <c>everyone</c> and welcome to the show!

00:00:05.100 --> 00:00:09.800
Today we are going to learn how to clip videos in Go.
`

	entries, err := ParseVTT(sampleVTT)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].Start != "00:00:01.000" || entries[0].End != "00:00:04.500" {
		t.Errorf("entry 0 timestamp mismatch: start=%s end=%s", entries[0].Start, entries[0].End)
	}

	if entries[0].Text != "Hello everyone and welcome to the show!" {
		t.Errorf("entry 0 text mismatch: %s", entries[0].Text)
	}
}
