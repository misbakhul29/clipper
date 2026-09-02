package transcriber

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetSubtitlePreset(t *testing.T) {
	t.Run("Hormozi preset", func(t *testing.T) {
		p := GetSubtitlePreset("hormozi", true)
		if p.Name != "hormozi" {
			t.Errorf("expected hormozi, got %s", p.Name)
		}
		if p.PrimaryColor != "&H0000FFFF" {
			t.Errorf("expected yellow &H0000FFFF, got %s", p.PrimaryColor)
		}
		if !strings.Contains(p.TransformTag, "fscx115") {
			t.Errorf("expected bounce transform tag, got %s", p.TransformTag)
		}
		if p.MaxWords != 2 {
			t.Errorf("expected 2 words, got %d", p.MaxWords)
		}
	})

	t.Run("Minimal and Devon alias", func(t *testing.T) {
		p1 := GetSubtitlePreset("minimal", true)
		p2 := GetSubtitlePreset("devon", true)
		if p1.Name != "minimal" || p2.Name != "minimal" {
			t.Errorf("expected minimal for both, got %s and %s", p1.Name, p2.Name)
		}
		if p1.PrimaryColor != "&H00FFFFFF" {
			t.Errorf("expected white primary color, got %s", p1.PrimaryColor)
		}
	})

	t.Run("Neon preset", func(t *testing.T) {
		p := GetSubtitlePreset("neon", true)
		if p.Name != "neon" {
			t.Errorf("expected neon, got %s", p.Name)
		}
		if p.PrimaryColor != "&H00FFFF00" {
			t.Errorf("expected cyan &H00FFFF00, got %s", p.PrimaryColor)
		}
		if !strings.Contains(p.TransformTag, "blur") {
			t.Errorf("expected blur tag, got %s", p.TransformTag)
		}
	})

	t.Run("Cinematic preset", func(t *testing.T) {
		p := GetSubtitlePreset("cinematic", false)
		if p.Name != "cinematic" {
			t.Errorf("expected cinematic, got %s", p.Name)
		}
		if p.FontName != "Georgia" {
			t.Errorf("expected Georgia font, got %s", p.FontName)
		}
		if p.MaxWords != 7 {
			t.Errorf("expected 7 max words, got %d", p.MaxWords)
		}
	})
}

func TestExportPresetASS(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clipper_preset_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	entries := []SubtitleEntry{
		{Start: "0:00:01.00", End: "0:00:04.00", Text: "Hello this is a viral clip for tiktok"},
	}

	outHormozi := filepath.Join(tmpDir, "hormozi.ass")
	if err := ExportPresetASS(entries, outHormozi, "hormozi", 60, true, "Montserrat-Bold", "strip"); err != nil {
		t.Fatalf("ExportPresetASS hormozi failed: %v", err)
	}

	data, err := os.ReadFile(outHormozi)
	if err != nil {
		t.Fatalf("failed to read exported ASS: %v", err)
	}
	content := string(data)

	// Check custom font and size override
	if !strings.Contains(content, "Montserrat-Bold") {
		t.Errorf("expected Montserrat-Bold in ASS header, content: %s", content)
	}
	if !strings.Contains(content, ",60,") {
		t.Errorf("expected font size 60 in ASS header, content: %s", content)
	}
	// Check pop-in animation tag
	if !strings.Contains(content, `{\fscx115\fscy115`) {
		t.Errorf("expected pop-in bounce tag in dialogue, content: %s", content)
	}
	// Check 2-word micro-chunking
	if !strings.Contains(content, "HELLO THIS") {
		t.Errorf("expected 2-word chunk 'HELLO THIS', content: %s", content)
	}
	// Check PlayRes for shorts (1080x1920)
	if !strings.Contains(content, "PlayResX: 1080") || !strings.Contains(content, "PlayResY: 1920") {
		t.Errorf("expected 1080x1920 for shorts canvas, content: %s", content)
	}

	// Test non-shorts canvas (1920x1080)
	outLandscape := filepath.Join(tmpDir, "landscape.ass")
	if err := ExportPresetASS(entries, outLandscape, "minimal", 40, false, "", "strip"); err != nil {
		t.Fatalf("ExportPresetASS landscape failed: %v", err)
	}
	dataLand, _ := os.ReadFile(outLandscape)
	contentLand := string(dataLand)
	if !strings.Contains(contentLand, "PlayResX: 1920") || !strings.Contains(contentLand, "PlayResY: 1080") {
		t.Errorf("expected 1920x1080 for landscape canvas, content: %s", contentLand)
	}
}

func TestParseVTT_Formats(t *testing.T) {
	vttSample := `WEBVTT
Kind: captions
Language: en

00:01.500 --> 00:03.200 line:85% position:50%
First cue without hours

00:00:05.100 --> 00:00:08.500
Second cue with standard hours

00:00:10,200 --> 00:00:12,800
Third cue with SRT style comma
`
	entries, err := ParseVTT(vttSample)
	if err != nil {
		t.Fatalf("unexpected error parsing VTT: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	if entries[0].Text != "First cue without hours" {
		t.Errorf("entry 0 text mismatch: %s", entries[0].Text)
	}
	if entries[0].Start != "00:00:01.500" || entries[0].End != "00:00:03.200" {
		t.Errorf("entry 0 timestamp mismatch: %+v", entries[0])
	}

	if entries[1].Text != "Second cue with standard hours" {
		t.Errorf("entry 1 text mismatch: %s", entries[1].Text)
	}

	if entries[2].Text != "Third cue with SRT style comma" {
		t.Errorf("entry 2 text mismatch: %s", entries[2].Text)
	}
	if entries[2].Start != "00:00:10.200" || entries[2].End != "00:00:12.800" {
		t.Errorf("entry 2 timestamp mismatch: %+v", entries[2])
	}
}

func TestExtractSDHAndSpeech(t *testing.T) {
	t.Run("Mixed speech and silent narrator", func(t *testing.T) {
		input := "bukan cuma merenggut akal sehat tapi juga nalar.\n[Laboratorium berlumuran darah akibat serangan zombi]"
		speech, sdh := ExtractSDHAndSpeech(input)

		expectedSpeech := "bukan cuma merenggut akal sehat tapi juga nalar."
		expectedSDH := "Laboratorium berlumuran darah akibat serangan zombi"

		if speech != expectedSpeech {
			t.Errorf("speech mismatch, got: '%s', want: '%s'", speech, expectedSpeech)
		}
		if sdh != expectedSDH {
			t.Errorf("sdh mismatch, got: '%s', want: '%s'", sdh, expectedSDH)
		}
	})

	t.Run("100% silent narrator", func(t *testing.T) {
		input := "[Lima peneliti yang terjebak di dalamnya]"
		speech, sdh := ExtractSDHAndSpeech(input)

		if speech != "" {
			t.Errorf("expected empty speech, got: '%s'", speech)
		}
		if sdh != "Lima peneliti yang terjebak di dalamnya" {
			t.Errorf("unexpected sdh: '%s'", sdh)
		}
	})

	t.Run("Clean speech without brackets", func(t *testing.T) {
		input := "Ini adalah murni ucapan narator aktif."
		speech, sdh := ExtractSDHAndSpeech(input)

		if speech != input {
			t.Errorf("speech mismatch: '%s'", speech)
		}
		if sdh != "" {
			t.Errorf("expected empty sdh, got: '%s'", sdh)
		}
	})
}

func TestExportPresetASS_SDHStrip(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clipper_sdh_strip_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	entries := []SubtitleEntry{
		{Start: "0:00:01.00", End: "0:00:03.00", Text: "Mereka yang terinfeksi\n[Laboratorium berlumuran darah]"},
		{Start: "0:00:04.00", End: "0:00:07.00", Text: "[Lima peneliti yang terjebak]"},
	}

	outFile := filepath.Join(tmpDir, "strip.ass")
	if err := ExportPresetASS(entries, outFile, "hormozi", 54, true, "", "strip"); err != nil {
		t.Fatalf("ExportPresetASS strip failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	content := string(data)

	// Should contain spoken dialogue
	if !strings.Contains(content, "MEREKA YANG") || !strings.Contains(content, "TERINFEKSI") {
		t.Errorf("expected speech words in output, got: %s", content)
	}
	// Should NOT contain bracketed text
	if strings.Contains(content, "Laboratorium") || strings.Contains(content, "peneliti") {
		t.Errorf("expected SDH to be stripped, got: %s", content)
	}
	// Pure SDH cue should be completely discarded (entry 0 has 3 words chunked into 2 dialogue lines with maxWords=2)
	if strings.Count(content, "Dialogue:") != 2 {
		t.Errorf("expected exactly 2 dialogue lines after chunking entry 0 and stripping pure SDH cue, got %d", strings.Count(content, "Dialogue:"))
	}
}

func TestExportPresetASS_SDHTopBox(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clipper_sdh_topbox_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	entries := []SubtitleEntry{
		{Start: "0:00:01.00", End: "0:00:03.00", Text: "Mereka yang terinfeksi\n[Laboratorium berlumuran darah]"},
		{Start: "0:00:04.00", End: "0:00:07.00", Text: "[Lima peneliti yang terjebak]"},
	}

	outFile := filepath.Join(tmpDir, "topbox.ass")
	if err := ExportPresetASS(entries, outFile, "hormozi", 54, true, "", "top-box"); err != nil {
		t.Fatalf("ExportPresetASS top-box failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	content := string(data)

	// Should have Narrator style defined
	if !strings.Contains(content, "Style: Narrator") {
		t.Errorf("expected Narrator style header, got: %s", content)
	}
	// Should have top-center alignment (8)
	if !strings.Contains(content, ",8,60,60,") {
		t.Errorf("expected alignment 8 for Narrator style, got: %s", content)
	}
	// Should render SDH in Narrator style
	if !strings.Contains(content, "Narrator,,0,0,0,,[Laboratorium berlumuran darah]") {
		t.Errorf("expected Narrator dialogue line, got: %s", content)
	}
	if !strings.Contains(content, "Narrator,,0,0,0,,[Lima peneliti yang terjebak]") {
		t.Errorf("expected pure SDH rendered in Narrator style, got: %s", content)
	}
	// Speech should still be rendered with Default style
	if !strings.Contains(content, "Default,,0,0,0,") {
		t.Errorf("expected Default dialogue line for speech, got: %s", content)
	}
}

func TestFilterSDHEntries(t *testing.T) {
	entries := []SubtitleEntry{
		{Start: "0:00:01.00", End: "0:00:03.00", Text: "Bukan cuma merenggut akal sehat\n[Laboratorium berlumuran darah]"},
		{Start: "0:00:03.50", End: "0:00:06.00", Text: "[Suara lonceng kencang]"},
	}

	// Strip mode should clean entry 0 and drop entry 1
	stripped := FilterSDHEntries(entries, "strip")
	if len(stripped) != 1 {
		t.Fatalf("expected 1 entry after strip, got %d", len(stripped))
	}
	if stripped[0].Text != "Bukan cuma merenggut akal sehat" {
		t.Errorf("unexpected stripped text: %s", stripped[0].Text)
	}

	// Top-box or keep should preserve all entries
	topBox := FilterSDHEntries(entries, "top-box")
	if len(topBox) != 2 {
		t.Fatalf("expected 2 entries for top-box, got %d", len(topBox))
	}
}

func TestExportPresetASS_EmptyEntriesError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clipper_sdh_empty_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	entries := []SubtitleEntry{
		{Start: "0:00:01.00", End: "0:00:03.00", Text: "[Hanya suara musik tegang]"},
	}

	outFile := filepath.Join(tmpDir, "empty.ass")
	err = ExportPresetASS(entries, outFile, "hormozi", 54, true, "", "strip")
	if err == nil {
		t.Errorf("expected error when all entries are stripped SDH, got nil")
	}
}

func TestExportPresetASS_TimestampFormatting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clipper_ts_format_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	entries := []SubtitleEntry{
		{Start: "01:23.456", End: "01:25.123", Text: "Single short word"},
	}

	outFile := filepath.Join(tmpDir, "formatted.ass")
	if err := ExportPresetASS(entries, outFile, "minimal", 40, false, "", "keep"); err != nil {
		t.Fatalf("ExportPresetASS failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	content := string(data)

	// Must be standard ASS format H:MM:SS.CC (e.g. 0:01:23.46)
	if !strings.Contains(content, "Dialogue: 0,0:01:23.46,0:01:25.12,Default,,0,0,0,,Single short word") {
		t.Errorf("expected normalized ASS timestamps 0:01:23.46, got: %s", content)
	}
}

func TestCleanSubtitleText(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{
			input:    `<c.colorE5E5E5>bukan cuma merenggut akal sehat &amp; nalar</c>`,
			expected: `bukan cuma merenggut akal sehat & nalar`,
		},
		{
			input:    `&quot;don&#39;t worry&quot; &lt;everyone&gt;`,
			expected: `"don't worry" <everyone>`,
		},
		{
			input:    "kata\u00a0pertama   kata\u00a0kedua",
			expected: "kata pertama kata kedua",
		},
		{
			input:    `<00:00:01.500><c>halo</c> <00:00:02.000><c>dunia</c>`,
			expected: `halo dunia`,
		},
	}

	for _, c := range cases {
		got := cleanSubtitleText(c.input)
		if got != c.expected {
			t.Errorf("cleanSubtitleText(%q) = %q, want %q", c.input, got, c.expected)
		}
	}
}
