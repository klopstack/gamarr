package download

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

func cryptoReader() io.Reader {
	return rand.Reader
}

// sanitizeFilename reduces an externally supplied name (Content-Disposition
// header, URL segment, platform slug, …) to a single safe path component:
// no separators, no traversal. Falls back to "download" if nothing is left.
// The filepath.IsLocal gate is what makes the result safe to join onto a
// trusted base directory.
func sanitizeFilename(name string) string {
	name = decodePercentName(name)
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" || name == "." || !filepath.IsLocal(name) {
		return "download"
	}
	return name
}

// decodePercentName unescapes %XX sequences (Myrient URL path segments).
// A literal "100% Cotton" has no hex escape and is left alone.
var pctHexRe = regexp.MustCompile(`%[0-9A-Fa-f]{2}`)

func decodePercentName(name string) string {
	if !pctHexRe.MatchString(name) {
		return name
	}
	if dec, err := url.PathUnescape(name); err == nil && dec != "" {
		return dec
	}
	return name
}

func destIsExistingFile(p string) bool {
	fi, err := os.Lstat(p)
	return err == nil && !fi.IsDir()
}

func isHTMLContentType(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	return ct == "text/html" || ct == "application/xhtml+xml"
}

func isHTMLPayload(b []byte) bool {
	s := bytes.TrimSpace(b)
	if len(s) == 0 {
		return false
	}
	if len(s) > 64 {
		s = s[:64]
	}
	ls := bytes.ToLower(s)
	return bytes.HasPrefix(ls, []byte("<!doctype html")) || bytes.HasPrefix(ls, []byte("<html"))
}

func looksLikeHTMLFile(p string) bool {
	fi, err := os.Stat(p)
	if err != nil || fi.IsDir() {
		return false
	}
	f, err := os.Open(p)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	return isHTMLPayload(buf[:n])
}

// isMyrientDirectoryURL reports a Myrient listing/index URL, which returns
// the site HTML page rather than a ROM. File URLs keep an archive extension.
func isMyrientDirectoryURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	if !strings.Contains(strings.ToLower(u.Host), "myrient") {
		return false
	}
	p := u.Path
	if p == "" || strings.HasSuffix(p, "/") {
		return true
	}
	name := decodePercentName(path.Base(p))
	return !isROMArchiveExt(path.Ext(name))
}

func isROMArchiveExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".zip", ".7z", ".rar", ".rvz", ".iso", ".chd", ".wbfs", ".gcz",
		".cso", ".nsp", ".xci", ".nsz", ".cia", ".nds", ".gba", ".gb", ".gbc",
		".nes", ".sfc", ".smc", ".n64", ".z64", ".v64", ".cue", ".bin",
		".pbp", ".wad", ".wux":
		return true
	default:
		return false
	}
}

// safeChild joins name onto dir and guarantees the result stays inside dir,
// defeating path traversal via crafted names. name may itself contain
// separators (a relative subpath) as long as it stays local.
func safeChild(dir, name string) (string, error) {
	if !filepath.IsLocal(name) {
		return "", fmt.Errorf("unsafe path %q escapes %q", name, dir)
	}
	return filepath.Join(dir, name), nil
}

// sanitizeLog strips newlines from externally supplied values before they
// reach the log, preventing forged log entries.
func sanitizeLog(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.ReplaceAll(s, "\r", " ")
}

// titlesMatch reports whether a tracked job title and a torrent name refer to
// the same release. Trackers often rename torrents, so it matches
// case-insensitively when either string contains the other.
func titlesMatch(title, torrentName string) bool {
	if title == "" || torrentName == "" {
		return false
	}
	a := strings.ToLower(title)
	b := strings.ToLower(torrentName)
	return strings.Contains(a, b) || strings.Contains(b, a)
}

// JobMatchesTorrent reports whether a tracked job refers to this torrent. Jobs
// recorded with an infohash match on that alone; rows from before it was stored
// fall back to the title. Shared by Manager.watchGameTorrent, Watcher.hasMatchingJob
// and the downloads API so they cannot disagree - a disagreement lets the watcher
// double-import a job's torrent, and made the API list one download twice.
func JobMatchesTorrent(infoHash, title, torrentHash, torrentName string) bool {
	if infoHash != "" {
		return strings.EqualFold(infoHash, torrentHash)
	}
	return titlesMatch(title, torrentName)
}

// jobForTorrent returns the id of the job already tracking this torrent. It
// shares JobMatchesTorrent with the watcher so the two cannot disagree about
// which row belongs to a torrent.
func (m *Manager) infoHashTracked(hash string) bool {
	if hash == "" {
		return false
	}
	for _, item := range m.jobs.Items() {
		ih, _ := item.Data["info_hash"].(string)
		if strings.EqualFold(ih, hash) {
			return true
		}
	}
	return false
}

func (m *Manager) jobForTorrent(torrentHash, torrentName string) (string, bool) {
	for _, item := range m.jobs.Items() {
		infoHash, _ := item.Data["info_hash"].(string)
		title, _ := item.Data["title"].(string)
		if JobMatchesTorrent(infoHash, title, torrentHash, torrentName) {
			return item.ID, true
		}
	}
	return "", false
}
