package sources

import (
	"encoding/json"
	"testing"
)

func TestPlatformPathList_UnmarshalJSON(t *testing.T) {
	var single PlatformPathList
	if err := json.Unmarshal([]byte(`"No-Intro/foo/"`), &single); err != nil {
		t.Fatal(err)
	}
	if single.Primary() != "No-Intro/foo/" {
		t.Errorf("single = %v", single)
	}

	var multi PlatformPathList
	if err := json.Unmarshal([]byte(`["disk/","tape/"]`), &multi); err != nil {
		t.Fatal(err)
	}
	if len(multi) != 2 || multi[0] != "disk/" || multi[1] != "tape/" {
		t.Errorf("multi = %v", multi)
	}

	var bad PlatformPathList
	if err := json.Unmarshal([]byte(`42`), &bad); err == nil {
		t.Error("expected error for non-string/non-array")
	}
}

func TestMinervaSpec_ResolveSlug(t *testing.T) {
	m := MinervaSpec{
		PlatformPaths: map[string]PlatformPathList{
			"ngc":  {"Redump/Nintendo - GameCube/"},
			"mame": {"MAME/ROMs (merged)/"},
			"c64": {
				"No-Intro/Commodore - Commodore 64/",
				"No-Intro/Commodore - Commodore 64 (Tapes)/",
			},
			"vic-20": {"No-Intro/Commodore - VIC-20/"},
		},
		PlatformAliases: map[string]string{
			"gamecube": "ngc",
			"arcade":   "mame",
		},
	}

	if canon, paths, ok := m.ResolveSlug("ngc"); !ok || canon != "ngc" || paths.Primary() != "Redump/Nintendo - GameCube/" {
		t.Errorf("direct ngc = %q %v %v", canon, paths, ok)
	}
	if canon, paths, ok := m.ResolveSlug("gamecube"); !ok || canon != "ngc" || paths.Primary() != "Redump/Nintendo - GameCube/" {
		t.Errorf("alias gamecube = %q %v %v", canon, paths, ok)
	}
	if canon, paths, ok := m.ResolveSlug("arcade"); !ok || canon != "mame" || paths.Primary() != "MAME/ROMs (merged)/" {
		t.Errorf("alias arcade = %q %v %v", canon, paths, ok)
	}
	if canon, _, ok := m.ResolveSlug("C64"); !ok || canon != "c64" {
		t.Errorf("upcased C64 = %q %v", canon, ok)
	}
	if canon, _, ok := m.ResolveSlug("vic20"); !ok || canon != "vic-20" {
		t.Errorf("unhyphenated vic20 = %q %v", canon, ok)
	}
	if _, _, ok := m.ResolveSlug("pc"); ok {
		t.Error("unknown slug should not resolve")
	}
	if _, _, ok := m.ResolveSlug(""); ok {
		t.Error("empty slug should not resolve")
	}
}
