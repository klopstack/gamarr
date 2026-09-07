package search

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"gamarr/internal/models"
	"gamarr/internal/platform"
	"gamarr/internal/sources"
)

const minervaMinQuery = 3

// magnetHashRe pulls the infohash out of a magnet:?xt=urn:btih:... link.
var magnetHashRe = regexp.MustCompile(`(?i)urn:btih:([a-fA-F0-9]{40}|[a-zA-Z2-7]{32})`)

type minervaHit struct {
	ID       int64  `json:"id"`
	FullPath string `json:"full_path"`
	Magnet   string `json:"magnet"`
}

// MinervaPlatformSlugs returns all slugs Minerva supports per the runtime
// sources registry.
func MinervaPlatformSlugs(reg *sources.Registry) []string {
	if reg == nil {
		return nil
	}
	slugs := make([]string, 0, len(reg.Minerva.PlatformConsoles))
	for s := range reg.Minerva.PlatformConsoles {
		slugs = append(slugs, s)
	}
	return slugs
}

// SearchMinerva searches Minerva Archive's JSON ROM index.
//
// Minerva is a searchable catalog over the same file tree Myrient hosts.
// Hits come back as paths; the download URL is built from the Myrient base
// so a result can be fetched as a single file instead of the collection
// torrent the catalog also advertises.
func SearchMinerva(reg *sources.Registry, query string, platformSlug string) []*models.SearchResult {
	if reg == nil || strings.TrimSpace(reg.Minerva.BaseURL) == "" {
		return nil
	}
	if IsCircuitOpen("minerva") {
		slog.Warn("minerva circuit open, skipping search")
		return nil
	}

	query = strings.TrimSpace(query)
	if len(query) < minervaMinQuery {
		return nil
	}
	if platformSlug != "" {
		if _, ok := reg.Minerva.PlatformConsoles[platformSlug]; !ok {
			return nil
		}
	}

	params := url.Values{"query": {strings.ToLower(query)}}
	console := ""
	if platformSlug != "" {
		console = reg.Minerva.PlatformConsoles[platformSlug]
		params.Set("console", console)
	}

	searchURL, err := url.JoinPath(strings.TrimRight(reg.Minerva.BaseURL, "/"), "/v1/api/rom/search")
	if err != nil {
		RecordSearchFail("minerva", err.Error())
		return nil
	}

	// Minerva's index is a full-catalog scan; a console-filtered title
	// search commonly takes 30–60s. The host is also slow to shake TLS
	// (Go's default handshake budget is 10s and misses it).
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSHandshakeTimeout = 30 * time.Second
	client := &http.Client{Timeout: 75 * time.Second, Transport: transport}
	req, err := http.NewRequest("GET", searchURL+"?"+params.Encode(), nil)
	if err != nil {
		RecordSearchFail("minerva", err.Error())
		return nil
	}
	req.Header.Set("User-Agent", "Gamarr/1.0")

	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("Minerva search error", "error", err)
		RecordSearchFail("minerva", err.Error())
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		RecordSearchFail("minerva", fmt.Sprintf("HTTP %d", resp.StatusCode))
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		RecordSearchFail("minerva", err.Error())
		return nil
	}

	var hits []minervaHit
	if err := json.Unmarshal(body, &hits); err != nil {
		RecordSearchFail("minerva", "invalid JSON: "+err.Error())
		return nil
	}
	if len(hits) > 100 {
		hits = hits[:100]
	}

	var results []*models.SearchResult
	for _, hit := range hits {
		title := minervaTitle(hit.FullPath)
		if title == "" {
			continue
		}
		if nonEnglishRegionRe.MatchString(title) && !englishRegionRe.MatchString(title) {
			continue
		}

		slug := platformSlug
		consoleName := console
		if slug == "" {
			slug = minervaSlugFromPath(hit.FullPath, reg.Minerva.PlatformConsoles)
			if slug != "" {
				consoleName = reg.Minerva.PlatformConsoles[slug]
			}
		}
		platName, isPC := minervaPlatformInfo(slug, consoleName)

		results = append(results, &models.SearchResult{
			Title:          title,
			SizeHuman:      "?",
			Indexer:        "Minerva",
			DownloadURL:    minervaFileURL(reg.Myrient.BaseURL, hit.FullPath),
			MagnetURL:      hit.Magnet,
			InfoHash:       minervaInfoHash(hit.Magnet),
			GUID:           minervaGUID(reg.Minerva.BaseURL, hit.ID),
			Platform:       platName,
			PlatformSlug:   slug,
			IsPC:           isPC,
			SourceType:     "ddl",
			SafetyScore:    95,
			SafetyWarnings: []string{},
		})
		if len(results) >= 20 {
			break
		}
	}

	RecordSearchSuccess("minerva")
	return results
}

func minervaTitle(fullPath string) string {
	name := path.Base(strings.ReplaceAll(strings.TrimSpace(fullPath), "\\", "/"))
	if name == "" || name == "." || name == "/" {
		return ""
	}
	return name
}

func minervaGUID(base string, id int64) string {
	if id <= 0 {
		return ""
	}
	return strings.TrimRight(base, "/") + "/rom?id=" + fmt.Sprintf("%d", id)
}

func minervaInfoHash(magnet string) string {
	m := magnetHashRe.FindStringSubmatch(magnet)
	if m == nil {
		return ""
	}
	return strings.ToLower(m[1])
}

// minervaFileURL maps a catalog path onto the Myrient file tree the listing
// describes. Segment-escape so spaces and brackets survive as a single URL.
func minervaFileURL(myrientBase, fullPath string) string {
	p := strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(fullPath), "\\", "/"), "./")
	p = strings.TrimPrefix(p, "/")
	if myrientBase == "" || p == "" {
		return ""
	}
	parts := strings.Split(p, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.TrimRight(myrientBase, "/") + "/" + strings.Join(parts, "/")
}

func minervaSlugFromPath(fullPath string, consoles map[string]string) string {
	haystack := minervaCompact(fullPath)
	bestSlug, bestLen := "", 0
	for slug, name := range consoles {
		if name == "" {
			continue
		}
		needle := minervaCompact(name)
		if n := len(needle); n > bestLen && strings.Contains(haystack, needle) {
			bestSlug, bestLen = slug, n
		}
	}
	return bestSlug
}

// minervaCompact folds "Nintendo - Game Boy Advance" and "Nintendo Game Boy
// Advance" to the same token string so path inference matches both catalogs.
func minervaCompact(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", " ")
	return strings.Join(strings.Fields(s), " ")
}

func minervaPlatformInfo(slug, consoleFallback string) (string, bool) {
	if slug != "" {
		for _, info := range platform.PlatformMap {
			if info.Slug == slug {
				return info.Name, info.IsPC
			}
		}
		for _, ep := range platform.ExtraPlatforms {
			if ep.Slug == slug {
				return ep.Name, false
			}
		}
	}
	if consoleFallback != "" {
		return consoleFallback, false
	}
	return "Unknown", false
}
