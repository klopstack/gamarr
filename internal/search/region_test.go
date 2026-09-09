package search

import (
	"testing"

	"gamarr/internal/models"
)

func TestRegionPreferenceRank_USAOverEurope(t *testing.T) {
	prefs := NewRegionPreferences([]string{"USA", "World", "Europe"}, nil)
	eu, _ := regionPreferenceRank("Super Mario World (Europe).zip", prefs)
	us, _ := regionPreferenceRank("Super Mario World (USA).zip", prefs)
	if us >= eu {
		t.Fatalf("USA rank %d should beat Europe rank %d", us, eu)
	}
}

func TestRegionPreferenceRank_DualRegion(t *testing.T) {
	prefs := NewRegionPreferences([]string{"USA", "Europe"}, nil)
	r, _ := regionPreferenceRank("Metroid (Japan, USA).zip", prefs)
	if r != 0 {
		t.Fatalf("dual-region with USA want rank 0, got %d", r)
	}
}

func TestRegionPreferenceRank_Untagged(t *testing.T) {
	prefs := NewRegionPreferences([]string{"USA"}, nil)
	r, _ := regionPreferenceRank("Untagged Game.zip", prefs)
	if r <= 0 {
		t.Fatalf("untagged want rank > 0, got %d", r)
	}
}

func TestRegionPreferenceRank_Language(t *testing.T) {
	prefs := NewRegionPreferences(nil, []string{"En", "English"})
	_, en := regionPreferenceRank("Game (Europe) (En,Fr,De).zip", prefs)
	_, fr := regionPreferenceRank("Game (Europe) (Fr,De).zip", prefs)
	if en >= fr {
		t.Fatalf("En rank %d should beat Fr-only rank %d", en, fr)
	}
}

func TestSortByScore_PrefersUSAOnTie(t *testing.T) {
	prefs := NewRegionPreferences([]string{"USA", "World", "Europe"}, nil)
	results := []*models.SearchResult{
		{Title: "Chrono Trigger (Europe).zip", Score: 80, Indexer: "Minerva"},
		{Title: "Chrono Trigger (USA).zip", Score: 80, Indexer: "Minerva"},
	}
	ScoreResults(results, "Chrono Trigger", "snes", prefs)
	SortByScore(results, prefs)
	if results[0].Title != "Chrono Trigger (USA).zip" {
		t.Fatalf("first result = %q, want USA", results[0].Title)
	}
}

func TestSortByScore_MinervaBeforeRegion(t *testing.T) {
	prefs := NewRegionPreferences([]string{"USA"}, nil)
	results := []*models.SearchResult{
		{Title: "Game (USA).zip", Score: 90, Indexer: "Prowlarr"},
		{Title: "Game (Europe).zip", Score: 70, Indexer: "Minerva"},
	}
	SortByScore(results, prefs)
	if results[0].Indexer != "Minerva" {
		t.Fatalf("first indexer = %q, want Minerva", results[0].Indexer)
	}
}

func TestSortByScore_DisabledPrefsPreservesOrder(t *testing.T) {
	results := []*models.SearchResult{
		{Title: "Game (Europe).zip", Score: 80},
		{Title: "Game (USA).zip", Score: 80},
	}
	SortByScore(results, RegionPreferences{})
	if results[0].Title != "Game (Europe).zip" {
		t.Fatalf("stable order changed with prefs disabled")
	}
}
