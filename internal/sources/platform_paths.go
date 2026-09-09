package sources

import (
	"encoding/json"
	"fmt"
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

// ResolveSlug maps a wishlist/API slug to the canonical slug and ordered paths.
// Aliases are checked only when slug is not a direct platform_paths key.
func (m *MinervaSpec) ResolveSlug(slug string) (canonical string, paths PlatformPathList, ok bool) {
	if m == nil {
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
	return "", nil, false
}
