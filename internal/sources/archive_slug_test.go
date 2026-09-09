package sources

import "testing"

func TestSlugForArchivePath(t *testing.T) {
	r, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		rel  string
		slug string
	}{
		{"No-Intro/Commodore - VIC-20/Omega Race (USA).zip", "vic-20"},
		{"Minerva_Myrient/No-Intro/Nintendo - Game Boy/Trip World (Europe).zip", "gb"},
		{"No-Intro/Nintendo - Game Boy Advance/Foo.zip", "gba"},
		{"No-Intro/Nintendo - Nintendo 64 (BigEndian)/Bar.zip", "n64"},
		{"No-Intro/Nintendo - Super Nintendo Entertainment System/Baz.zip", "snes"},
		{"No-Intro/Sega - Mega Drive - Genesis/Quux.zip", "genesis"},
		{"No-Intro/Nintendo - Nintendo DS (Decrypted)/x.zip", "nds"},
		{"No-Intro/Nintendo - Nintendo Entertainment System (Headered)/y.zip", "nes"},
		{"Redump/Microsoft - Xbox/Halo (USA).zip", "xbox"},
		{"Redump/Sony - PlayStation/Wipeout.zip", "psx"},
		{"Redump/Sony - PlayStation Portable/Game.zip", "psp"},
		{"Redump/Nintendo - Wii - NKit RVZ [zstd-19-128k]/Wii Sports.rvz", "wii"},
		{"Minerva_Myrient", ""},
		{"No-Intro/Not A Real Collection/x.zip", ""},
	}
	for _, tc := range cases {
		got, ok := r.SlugForArchivePath(tc.rel)
		if tc.slug == "" {
			if ok {
				t.Errorf("SlugForArchivePath(%q) = %q, want none", tc.rel, got)
			}
			continue
		}
		if !ok || got != tc.slug {
			t.Errorf("SlugForArchivePath(%q) = %q %v, want %q", tc.rel, got, ok, tc.slug)
		}
	}
}
