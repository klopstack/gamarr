package search

import (
	"path"
	"regexp"
	"strings"
)

// RegionPreferences is an ordered list of preferred region and language tags
// extracted from No-Intro-style release names. Earlier entries win on ties.
type RegionPreferences struct {
	Regions   []string
	Languages []string
}

// NewRegionPreferences builds prefs from config slices (may be empty).
func NewRegionPreferences(regions, languages []string) RegionPreferences {
	return RegionPreferences{
		Regions:   normalizePrefList(regions),
		Languages: normalizePrefList(languages),
	}
}

func (p RegionPreferences) Enabled() bool {
	return len(p.Regions) > 0 || len(p.Languages) > 0
}

var tagTokenRe = regexp.MustCompile(`\(([^)]*)\)|\[([^\]]*)\]`)

var skipTagRe = regexp.MustCompile(
	`(?i)^(disc|disk|side|tape|cd|dvd|rev|revision|beta|proto|demo|sample|` +
		`kiosk|demo|promo|alt|hack|translation|unl|unlicensed|aftermarket)` +
		`(\s|\d|$)`,
)

// regionPreferenceRank returns a sort key; lower is better. When prefs are
// disabled every title returns 0 so tie-break is a no-op.
func regionPreferenceRank(title string, prefs RegionPreferences) (regionRank, langRank int) {
	if !prefs.Enabled() {
		return 0, 0
	}
	regionRank = rankAgainstPrefs(extractRegionTokens(title), prefs.Regions, len(prefs.Regions)+10, len(prefs.Regions)+20)
	if len(prefs.Languages) > 0 {
		langRank = rankAgainstPrefs(extractLanguageTokens(title), prefs.Languages, len(prefs.Languages)+5, len(prefs.Languages)+10)
	}
	return regionRank, langRank
}

func normalizePrefList(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := make(map[string]bool)
	for _, raw := range in {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		key := strings.ToLower(raw)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func rankAgainstPrefs(found, prefs []string, unknownRank, untaggedRank int) int {
	if len(prefs) == 0 {
		return 0
	}
	if len(found) == 0 {
		return untaggedRank
	}
	best := unknownRank
	for _, tok := range found {
		for i, pref := range prefs {
			if tokenMatchesPref(tok, pref) {
				if i < best {
					best = i
				}
				break
			}
		}
	}
	return best
}

func tokenMatchesPref(token, pref string) bool {
	token = strings.ToLower(strings.TrimSpace(token))
	pref = strings.ToLower(strings.TrimSpace(pref))
	if token == pref {
		return true
	}
	if canon, ok := canonicalRegion(token); ok {
		if prefCanon, ok := canonicalRegion(pref); ok {
			return canon == prefCanon
		}
	}
	if canon, ok := canonicalLanguage(token); ok {
		if prefCanon, ok := canonicalLanguage(pref); ok {
			return canon == prefCanon
		}
	}
	return false
}

func extractTagGroups(title string) []string {
	base := path.Base(title)
	var groups []string
	for _, m := range tagTokenRe.FindAllStringSubmatch(base, -1) {
		blob := m[1]
		if blob == "" {
			blob = m[2]
		}
		blob = strings.TrimSpace(blob)
		if blob != "" {
			groups = append(groups, blob)
		}
	}
	return groups
}

func extractRegionTokens(title string) []string {
	var out []string
	for _, blob := range extractTagGroups(title) {
		for _, tok := range splitTagBlob(blob) {
			if skipTagRe.MatchString(tok) {
				continue
			}
			if _, ok := canonicalRegion(tok); ok {
				out = append(out, tok)
			}
		}
	}
	return out
}

func extractLanguageTokens(title string) []string {
	var out []string
	for _, blob := range extractTagGroups(title) {
		// Language lists are usually comma-heavy (En,Fr,De); region tags rarely are.
		if !strings.Contains(blob, ",") {
			if _, ok := canonicalRegion(blob); ok {
				continue
			}
		}
		for _, tok := range splitTagBlob(blob) {
			if skipTagRe.MatchString(tok) {
				continue
			}
			if _, ok := canonicalRegion(tok); ok {
				continue
			}
			if _, ok := canonicalLanguage(tok); ok {
				out = append(out, tok)
			}
		}
	}
	return out
}

func splitTagBlob(blob string) []string {
	parts := strings.FieldsFunc(blob, func(r rune) bool {
		return r == ',' || r == '/' || r == ';'
	})
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func canonicalRegion(token string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(token)) {
	case "usa", "us", "u":
		return "usa", true
	case "world", "wor", "w":
		return "world", true
	case "europe", "eur", "eu", "e":
		return "europe", true
	case "japan", "jap", "jp", "j":
		return "japan", true
	case "korea", "kor", "kr", "k":
		return "korea", true
	case "china", "chn", "cn":
		return "china", true
	case "taiwan", "twn", "tw":
		return "taiwan", true
	case "australia", "aus", "au":
		return "australia", true
	case "canada", "can", "ca":
		return "canada", true
	case "brazil", "bra", "br":
		return "brazil", true
	case "asia":
		return "asia", true
	case "hong kong", "hongkong", "hk":
		return "hongkong", true
	default:
		return "", false
	}
}

func canonicalLanguage(token string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(token)) {
	case "en", "eng", "english":
		return "en", true
	case "fr", "fre", "fra", "french":
		return "fr", true
	case "de", "ger", "deu", "german":
		return "de", true
	case "es", "spa", "spanish":
		return "es", true
	case "it", "ita", "italian":
		return "it", true
	case "pt", "por", "portuguese":
		return "pt", true
	case "nl", "dut", "dutch":
		return "nl", true
	case "sv", "swe", "swedish":
		return "sv", true
	case "no", "nor", "norwegian":
		return "no", true
	case "da", "dan", "danish":
		return "da", true
	case "fi", "fin", "finnish":
		return "fi", true
	case "pl", "pol", "polish":
		return "pl", true
	case "ru", "rus", "russian":
		return "ru", true
	case "ja", "jpn", "japanese":
		return "ja", true
	case "ko", "kor", "korean":
		return "ko", true
	case "zh", "chi", "chinese":
		return "zh", true
	default:
		if strings.HasPrefix(strings.ToLower(token), "multi") {
			return "multi", true
		}
		return "", false
	}
}
