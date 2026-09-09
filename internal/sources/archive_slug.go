package sources

import (
	"path/filepath"
	"strings"
)

// SlugForArchivePath maps a Myrient/Minerva relative path to the RomM slug
// whose platform_paths prefix matches. Longest prefix wins so a more specific
// collection beats a shorter one. The torrent's internal root (Minerva_Myrient)
// is not a platform.
func (r *Registry) SlugForArchivePath(rel string) (string, bool) {
	if r == nil {
		return "", false
	}
	rel = normalizeArchiveRel(rel)
	if rel == "" {
		return "", false
	}
	bestSlug := ""
	bestLen := 0
	for slug, paths := range r.Minerva.PlatformPaths {
		for _, p := range paths {
			if n := archivePrefixLen(rel, p); n > bestLen {
				bestLen = n
				bestSlug = slug
			}
		}
	}
	for slug, p := range r.Myrient.PlatformPaths {
		if n := archivePrefixLen(rel, p); n > bestLen {
			bestLen = n
			bestSlug = slug
		}
	}
	return bestSlug, bestLen > 0
}

func normalizeArchiveRel(rel string) string {
	rel = filepath.ToSlash(rel)
	rel = strings.TrimPrefix(rel, "/")
	for {
		first, rest, ok := strings.Cut(rel, "/")
		if !ok {
			break
		}
		switch strings.ToLower(strings.TrimSpace(first)) {
		case "minerva_myrient", "minerva myrient", "minerva archive":
			rel = rest
			continue
		}
		break
	}
	return rel
}

func archivePrefixLen(rel, collection string) int {
	c := strings.Trim(filepath.ToSlash(collection), "/")
	if c == "" {
		return 0
	}
	if rel == c || strings.HasPrefix(rel, c+"/") {
		return len(c)
	}
	// No-Intro dump-type suffixes: "Commodore - Commodore 64 (PP)"
	if strings.HasPrefix(rel, c+" (") {
		return len(c)
	}
	return 0
}
