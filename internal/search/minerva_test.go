package search

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gamarr/internal/sources"
)

func minervaTestRegistry(t *testing.T, base string) *sources.Registry {
	t.Helper()
	reg := testRegistry(t)
	reg.Minerva.BaseURL = base
	reg.Myrient.BaseURL = "https://myrient.example.test/files/"
	return reg
}

func TestMinervaPlatformSlugs(t *testing.T) {
	reg := testRegistry(t)
	slugs := MinervaPlatformSlugs(reg)
	if len(slugs) == 0 {
		t.Fatal("expected non-empty slug list")
	}
	found := make(map[string]bool)
	for _, s := range slugs {
		found[s] = true
	}
	for _, want := range []string{"nes", "snes", "n64", "psx", "gba", "switch"} {
		if !found[want] {
			t.Errorf("expected slug %q in Minerva platforms", want)
		}
	}
}

func TestSearchMinerva_EmptyQueryAndShortQuery(t *testing.T) {
	reg := minervaTestRegistry(t, "http://127.0.0.1:1/")
	if results := SearchMinerva(reg, "", "snes"); results != nil {
		t.Error("empty query should yield nil")
	}
	if results := SearchMinerva(reg, "ab", "snes"); results != nil {
		t.Error("query shorter than 3 chars should yield nil")
	}
}

func TestSearchMinerva_UnknownPlatform(t *testing.T) {
	reg := minervaTestRegistry(t, "http://127.0.0.1:1/")
	if results := SearchMinerva(reg, "mario", "pc"); results != nil {
		t.Error("unmapped platform should yield nil")
	}
}

func TestSearchMinerva_EmptyRegistry(t *testing.T) {
	if results := SearchMinerva(nil, "mario", "snes"); results != nil {
		t.Error("nil registry should yield nil")
	}
	reg := &sources.Registry{}
	if results := SearchMinerva(reg, "mario", "snes"); results != nil {
		t.Error("empty base URL should yield nil")
	}
}

func TestSearchMinerva_ParsesResults(t *testing.T) {
	t.Cleanup(func() { RecordSearchSuccess("minerva") })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/api/rom/search" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("query"); got != "chrono trigger" {
			t.Errorf("query = %q", got)
		}
		if got := r.URL.Query().Get("console"); got != "Super Nintendo Entertainment System" {
			t.Errorf("console = %q", got)
		}
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"id":        823208,
				"full_path": "./No-Intro/Nintendo - Super Nintendo Entertainment System/Chrono Trigger (USA).zip",
				"magnet":    "magnet:?xt=urn:btih:C5C78AB0432DCC14FEC568BA5DAAD2D9CC98F4DC&dn=Minerva_Myrient",
			},
			{
				"id":        1,
				"full_path": "./TOSEC/Nintendo/Super Famicom/Games/Chrono Trigger (1995)(Square)(Japan).zip",
				"magnet":    "magnet:?xt=urn:btih:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
			{
				"id":        2,
				"full_path": "./No-Intro/Nintendo - Super Nintendo Entertainment System/Chrono Trigger (Europe).zip",
				"magnet":    "magnet:?xt=urn:btih:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			},
		})
	}))
	t.Cleanup(srv.Close)

	reg := minervaTestRegistry(t, srv.URL+"/")
	results := SearchMinerva(reg, "Chrono Trigger", "snes")
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2 (Japan dump dropped)", len(results))
	}
	r := results[0]
	if r.Indexer != "Minerva" || r.SourceType != "ddl" {
		t.Errorf("indexer/type = %q %q", r.Indexer, r.SourceType)
	}
	if r.Title != "Chrono Trigger (USA).zip" {
		t.Errorf("title = %q", r.Title)
	}
	if r.PlatformSlug != "snes" || r.Platform != "SNES" {
		t.Errorf("platform = %q %q", r.Platform, r.PlatformSlug)
	}
	if r.InfoHash != "c5c78ab0432dcc14fec568ba5daad2d9cc98f4dc" {
		t.Errorf("infohash = %q", r.InfoHash)
	}
	if r.GUID != srv.URL+"/rom?id=823208" {
		t.Errorf("guid = %q", r.GUID)
	}
	wantURL := "https://myrient.example.test/files/No-Intro/Nintendo%20-%20Super%20Nintendo%20Entertainment%20System/Chrono%20Trigger%20%28USA%29.zip"
	if r.DownloadURL != wantURL {
		t.Errorf("download URL = %q, want %q", r.DownloadURL, wantURL)
	}
	if results[1].Title != "Chrono Trigger (Europe).zip" {
		t.Errorf("second title = %q", results[1].Title)
	}
}

func TestSearchMinerva_InfersSlugFromPath(t *testing.T) {
	t.Cleanup(func() { RecordSearchSuccess("minerva") })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("console") != "" {
			t.Errorf("unfiltered search should omit console, got %q", r.URL.Query().Get("console"))
		}
		_ = json.NewEncoder(w).Encode([]minervaHit{{
			ID:       9,
			FullPath: "./No-Intro/Nintendo - Game Boy Advance/Metroid Fusion (USA).zip",
			Magnet:   "magnet:?xt=urn:btih:1111111111111111111111111111111111111111",
		}})
	}))
	t.Cleanup(srv.Close)

	reg := minervaTestRegistry(t, srv.URL+"/")
	results := SearchMinerva(reg, "metroid", "")
	if len(results) != 1 {
		t.Fatalf("got %d results", len(results))
	}
	if results[0].PlatformSlug != "gba" {
		t.Errorf("slug = %q, want gba from path", results[0].PlatformSlug)
	}
	if results[0].Platform != "Game Boy Advance" {
		t.Errorf("platform = %q", results[0].Platform)
	}
}

func TestSearchMinerva_HTTPError(t *testing.T) {
	t.Cleanup(func() { RecordSearchSuccess("minerva") })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)
	reg := minervaTestRegistry(t, srv.URL+"/")
	if results := SearchMinerva(reg, "mario", "snes"); len(results) != 0 {
		t.Errorf("HTTP 500 should yield no results, got %d", len(results))
	}
}

func TestMinervaFileURL(t *testing.T) {
	got := minervaFileURL("https://myrient.example.test/files/", "./No-Intro/Nintendo - SNES/Foo (USA).zip")
	want := "https://myrient.example.test/files/No-Intro/Nintendo%20-%20SNES/Foo%20%28USA%29.zip"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if minervaFileURL("", "x.zip") != "" || minervaFileURL("http://x/", "") != "" {
		t.Error("empty inputs should yield empty URL")
	}
}

func TestMinervaInfoHashAndTitle(t *testing.T) {
	if got := minervaInfoHash("magnet:?xt=urn:btih:AbCdEf0123456789AbCdEf0123456789AbCdEf01&dn=x"); got != "abcdef0123456789abcdef0123456789abcdef01" {
		t.Errorf("hash = %q", got)
	}
	if minervaInfoHash("not-a-magnet") != "" {
		t.Error("non-magnet should be empty")
	}
	if minervaTitle("./a/b/Game.zip") != "Game.zip" {
		t.Errorf("title = %q", minervaTitle("./a/b/Game.zip"))
	}
	if minervaTitle("./") != "" {
		t.Error("bare ./ should be empty")
	}
}

func TestSearchMinerva_CapsAt20(t *testing.T) {
	t.Cleanup(func() { RecordSearchSuccess("minerva") })
	hits := make([]minervaHit, 30)
	for i := range hits {
		hits[i] = minervaHit{
			ID:       int64(i + 1),
			FullPath: "./No-Intro/Nintendo - Super Nintendo Entertainment System/Game (USA).zip",
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(hits)
	}))
	t.Cleanup(srv.Close)
	reg := minervaTestRegistry(t, srv.URL+"/")
	results := SearchMinerva(reg, "game", "snes")
	if len(results) != 20 {
		t.Errorf("got %d results, want 20", len(results))
	}
}

func TestSearchMinerva_InvalidJSON(t *testing.T) {
	t.Cleanup(func() { RecordSearchSuccess("minerva") })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	t.Cleanup(srv.Close)
	reg := minervaTestRegistry(t, srv.URL+"/")
	if results := SearchMinerva(reg, "mario", "snes"); len(results) != 0 {
		t.Errorf("invalid JSON should yield no results, got %d", len(results))
	}
}

func TestMinervaGUID(t *testing.T) {
	if got := minervaGUID("https://minerva-archive.org/", 42); got != "https://minerva-archive.org/rom?id=42" {
		t.Errorf("guid = %q", got)
	}
	if minervaGUID("https://x/", 0) != "" {
		t.Error("id 0 should be empty")
	}
}

func TestMinervaSlugFromPathPrefersLongestConsoleName(t *testing.T) {
	consoles := map[string]string{
		"gb":  "Nintendo Game Boy",
		"gba": "Nintendo Game Boy Advance",
	}
	if got := minervaSlugFromPath("./No-Intro/Nintendo Game Boy Advance/x.zip", consoles); got != "gba" {
		t.Errorf("got %q, want gba (longer match)", got)
	}
	if got := minervaSlugFromPath("./No-Intro/Nintendo - Game Boy Advance/x.zip", consoles); got != "gba" {
		t.Errorf("hyphenated folder = %q, want gba", got)
	}
}
