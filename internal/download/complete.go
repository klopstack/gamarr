package download

import (
	"path/filepath"
	"strings"

	"gamarr/internal/qbit"
)

// DisplayProgress rounds a 0–1 fraction to the one-decimal percent the UI
// already uses for torrent progress, so a file bar and a torrent bar agree.
func DisplayProgress(p float64) float64 {
	return float64(int(p*1000)) / 10.0
}

// ArchiveShellTitle reports whether a job is the torrent-named wrapper orphan
// recovery used to mint for a shared archive magnet, not a ROM inside it.
func ArchiveShellTitle(jobTitle, torrentName string) bool {
	title := strings.TrimSpace(jobTitle)
	if title == "" {
		return true
	}
	if torrentName != "" && strings.EqualFold(title, torrentName) {
		return true
	}
	switch strings.ToLower(title) {
	case "minerva_myrient", "minerva myrient", "archive download":
		return true
	}
	return false
}

// TorrentDownloadComplete reports whether every file the client is actually
// fetching has finished. Skipped files (priority 0) do not count: qBittorrent
// often never reports torrent progress 1.0 or an UP state for a partial
// selection, which is every Minerva archive magnet.
func TorrentDownloadComplete(t qbit.Torrent, files []qbit.TorrentFile) bool {
	switch t.State {
	case "stoppedUP", "pausedUP", "stalledUP", "uploading", "forcedUP", "queuedUP":
		return true
	}
	if len(files) == 0 {
		return t.Progress >= 1.0
	}
	wanted := 0
	for _, f := range files {
		if f.Priority <= 0 {
			continue
		}
		wanted++
		if f.Progress < 1.0 {
			return false
		}
	}
	if wanted == 0 {
		return t.Progress >= 1.0
	}
	return true
}

// JobDownloadComplete reports whether this job's payload is on disk. A title
// that matches files inside the torrent is complete when those files are;
// skipped siblings are irrelevant. A title that matches nothing (a normal
// single-release torrent) falls back to the torrent's wanted set.
func JobDownloadComplete(t qbit.Torrent, files []qbit.TorrentFile, title string) bool {
	if matched := torrentFilesForTitle(files, title); len(matched) > 0 {
		for _, f := range matched {
			if f.Progress < 1.0 {
				return false
			}
		}
		return true
	}
	return TorrentDownloadComplete(t, files)
}

// WantedProgress is the size-weighted progress of priority>0 files, or false
// when the client has no selection to measure.
func WantedProgress(files []qbit.TorrentFile) (float64, bool) {
	var done, total float64
	for _, f := range files {
		if f.Priority <= 0 || f.Size <= 0 {
			continue
		}
		total += float64(f.Size)
		p := f.Progress
		if p < 0 {
			p = 0
		}
		if p > 1 {
			p = 1
		}
		done += float64(f.Size) * p
	}
	if total == 0 {
		return 0, false
	}
	return DisplayProgress(done / total), true
}

func torrentContentRoot(files []qbit.TorrentFile) string {
	root := ""
	for _, f := range files {
		first, _, found := strings.Cut(f.Name, "/")
		if !found {
			return ""
		}
		if root == "" {
			root = first + "/"
		} else if first+"/" != root {
			return ""
		}
	}
	return root
}

func torrentRelPath(files []qbit.TorrentFile, f qbit.TorrentFile) string {
	rel := f.Name
	if root := torrentContentRoot(files); root != "" {
		rel = strings.TrimPrefix(f.Name, root)
	}
	if rel == "" {
		rel = filepath.Base(f.Name)
	}
	return rel
}

func torrentFilesForTitle(files []qbit.TorrentFile, title string) []qbit.TorrentFile {
	idxs := matchTorrentFileIndexes(files, title)
	if len(idxs) == 0 {
		return nil
	}
	want := make(map[int]bool, len(idxs))
	for _, i := range idxs {
		want[i] = true
	}
	usePos := true
	for _, f := range files {
		if f.Index != 0 {
			usePos = false
			break
		}
	}
	usePos = usePos && len(files) > 1
	var out []qbit.TorrentFile
	for i, f := range files {
		idx := f.Index
		if usePos {
			idx = i
		}
		if want[idx] {
			out = append(out, f)
		}
	}
	return out
}

// jobSourcePaths maps a job title onto paths under the torrent's content
// folder. Empty means the title does not name files inside the torrent, so the
// caller should import the content root as it always has.
func jobSourcePaths(contentPath string, files []qbit.TorrentFile, title string) []string {
	matched := torrentFilesForTitle(files, title)
	if len(matched) == 0 {
		return nil
	}
	out := make([]string, 0, len(matched))
	for _, f := range matched {
		out = append(out, filepath.Join(contentPath, filepath.FromSlash(torrentRelPath(files, f))))
	}
	return out
}

func jobStatusActive(status string) bool {
	switch status {
	case "completed", "completed_unorganized", "error", "interrupted", "dead_letter":
		return false
	default:
		return true
	}
}
