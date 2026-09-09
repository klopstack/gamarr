package download

import (
	"testing"
)

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"strip zip extension", "Game.zip", "Game"},
		{"strip nsp extension", "Zelda.nsp", "Zelda"},
		{"strip rar extension", "Game.rar", "Game"},
		{"strip 7z extension", "Game.7z", "Game"},
		{"strip iso extension", "Game.iso", "Game"},
		{"URL decode spaces", "Super%20Mario%20Bros", "Super Mario Bros"},
		{"URL decode parens", "Game%28USA%29", "Game(USA)"},
		{"URL decode comma", "Game%2C Part 2", "Game, Part 2"},
		{"trim whitespace", "  Game  ", "Game"},
		{"no extension to strip", "Plain Game", "Plain Game"},
		{"case insensitive ext", "Game.ZIP", "Game"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanTitle(tt.in)
			if got != tt.want {
				t.Errorf("cleanTitle(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestDecodePercentName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Silent%20Hill%20(Europe).zip", "Silent Hill (Europe).zip"},
		{"Game%28USA%29%2C%20Rev.zip", "Game(USA), Rev.zip"},
		{"100% Cotton.zip", "100% Cotton.zip"},
		{"plain.zip", "plain.zip"},
	}
	for _, tc := range cases {
		if got := decodePercentName(tc.in); got != tc.want {
			t.Errorf("decodePercentName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsMyrientDirectoryURL(t *testing.T) {
	if !isMyrientDirectoryURL("https://myrient.erista.me/files/Redump/Nintendo - Wii - NKit RVZ [zstd-19-128k]/") {
		t.Error("collection listing should be rejected")
	}
	if !isMyrientDirectoryURL("https://myrient.erista.me/files/Redump/Nintendo%20-%20Wii/") {
		t.Error("encoded collection listing should be rejected")
	}
	if isMyrientDirectoryURL("https://myrient.erista.me/files/Redump/Nintendo%20-%20Wii/Silent%20Hill%20(Europe).zip") {
		t.Error("file URL should be allowed")
	}
	if isMyrientDirectoryURL("http://127.0.0.1:9/files/zelda.gba") {
		t.Error("non-Myrient test URLs must stay allowed")
	}
}

func TestIsHTMLPayload(t *testing.T) {
	if !isHTMLPayload([]byte("<!DOCTYPE html>\n<html><title>Myrient</title>")) {
		t.Error("doctype html should match")
	}
	if !isHTMLPayload([]byte("  <html lang=\"en\">Fast and Reliable")) {
		t.Error("<html prefix should match")
	}
	if isHTMLPayload([]byte("PK\x03\x04romzip")) {
		t.Error("zip magic must not match")
	}
}

func TestTitlesMatch(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		torrentName string
		want        bool
	}{
		{"exact", "Super Game", "Super Game", true},
		{"case insensitive", "Super Game", "SUPER game", true},
		{"torrent renamed with suffix", "Super Game", "Super Game (USA) [Repack]", true},
		{"title contains torrent name", "Super Game Deluxe Edition", "super game deluxe", true},
		{"unrelated", "Super Game", "Other Thing", false},
		{"empty title never matches", "", "Anything", false},
		{"empty torrent name never matches", "Super Game", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := titlesMatch(tt.title, tt.torrentName); got != tt.want {
				t.Errorf("titlesMatch(%q, %q) = %v, want %v", tt.title, tt.torrentName, got, tt.want)
			}
		})
	}
}

func TestJobMatchesTorrent(t *testing.T) {
	const terraria = "Terraria (v1.4.5.0 + Bonus OST, MULTi9) [FitGirl Repack]"

	tests := []struct {
		name                     string
		infoHash, title          string
		torrentHash, torrentName string
		want                     bool
	}{
		{"hash matches a renamed torrent", "abc123", terraria, "abc123", "Terraria [FitGirl Repack]", true},
		{"hash is case insensitive", "ABC123", terraria, "abc123", "Terraria [FitGirl Repack]", true},
		{"hash mismatch is not rescued by the title", "abc123", "Super Game", "def456", "Super Game", false},
		{"hashless job falls back to the title", "", "Super Game", "abc123", "Super Game (USA)", true},
		{"hashless job with an unrelated title", "", "Super Game", "abc123", "Other Thing", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JobMatchesTorrent(tt.infoHash, tt.title, tt.torrentHash, tt.torrentName)
			if got != tt.want {
				t.Errorf("JobMatchesTorrent(%q, %q, %q, %q) = %v, want %v",
					tt.infoHash, tt.title, tt.torrentHash, tt.torrentName, got, tt.want)
			}
		})
	}
}

func TestPlatformNameFromSlug(t *testing.T) {
	tests := []struct {
		slug string
		want string
	}{
		{"gba", "Game Boy Advance"},
		{"nes", "NES"},
		{"switch", "Switch"},
		{"psx", "PS1"},
		{"ngc", "GameCube"},
		{"unknown", "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			got := platformNameFromSlug(tt.slug)
			if got != tt.want {
				t.Errorf("platformNameFromSlug(%q) = %q, want %q", tt.slug, got, tt.want)
			}
		})
	}
}

func TestGameExtensions(t *testing.T) {
	expected := []string{".nsp", ".xci", ".nes", ".gba", ".iso", ".zip", ".exe"}
	for _, ext := range expected {
		if !gameExtensions[ext] {
			t.Errorf("expected %q in gameExtensions", ext)
		}
	}

	notExpected := []string{".txt", ".pdf", ".mp3", ".jpg"}
	for _, ext := range notExpected {
		if gameExtensions[ext] {
			t.Errorf("expected %q NOT in gameExtensions", ext)
		}
	}
}
