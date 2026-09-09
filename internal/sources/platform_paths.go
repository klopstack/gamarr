package sources

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PlatformPathList is one or more archive browse paths searched in order.
// The first path is preferred (e.g. disk); later entries are fallbacks (e.g. tape).
type PlatformPathList []string

func (p *PlatformPathList) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*p = nil
		return nil
	}
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		if single != "" {
			*p = PlatformPathList{single}
		} else {
			*p = nil
		}
		return nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("platform path: want string or []string: %w", err)
	}
	*p = PlatformPathList(list)
	return nil
}

// Primary returns the preferred (first-tier) browse path, or "".
func (p PlatformPathList) Primary() string {
	if len(p) == 0 {
		return ""
	}
	return p[0]
}

func compactSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "")
	return strings.ReplaceAll(s, "_", "")
}

// ResolveSlug maps a wishlist/API slug to the canonical slug and ordered paths.
// Card/prompt values are often upcased (C64) or unhyphenated (vic20).
func (m *MinervaSpec) ResolveSlug(slug string) (canonical string, paths PlatformPathList, ok bool) {
	if m == nil {
		return "", nil, false
	}
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return "", nil, false
	}
	if paths, ok = m.PlatformPaths[slug]; ok && paths.Primary() != "" {
		return slug, paths, true
	}
	if canon, aliased := m.PlatformAliases[slug]; aliased {
		if paths, ok = m.PlatformPaths[canon]; ok && paths.Primary() != "" {
			return canon, paths, true
		}
	}
	for key, p := range m.PlatformPaths {
		if strings.EqualFold(key, slug) && p.Primary() != "" {
			return key, p, true
		}
	}
	for alias, canon := range m.PlatformAliases {
		if strings.EqualFold(alias, slug) {
			if p, found := m.PlatformPaths[canon]; found && p.Primary() != "" {
				return canon, p, true
			}
		}
	}
	want := compactSlug(slug)
	var hit string
	var hitPaths PlatformPathList
	n := 0
	for key, p := range m.PlatformPaths {
		if compactSlug(key) == want && p.Primary() != "" {
			n++
			hit, hitPaths = key, p
		}
	}
	if n == 1 {
		return hit, hitPaths, true
	}
	return "", nil, false
}
