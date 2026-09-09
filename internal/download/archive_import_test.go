package download

import (
	"os"
	"path/filepath"
	"testing"

	"gamarr/internal/qbit"
	"gamarr/internal/sources"
)

func TestJobFileReadyArchiveMember(t *testing.T) {
	cfg := newTestConfig(t)
	jobs := newTestJobs(t)
	qm := newQbitMock(t)
	cfg.QBURL = qm.srv.URL
	m := New(cfg, jobs, qm.client())

	hash := "wii-hash"
	qm.setFiles([]qbit.TorrentFile{
		{Name: "Minerva_Myrient/Redump/Wii/Animal Crossing.zip", Priority: 1, Progress: 1.0, Index: 0},
		{Name: "Minerva_Myrient/Redump/Wii/Other.zip", Priority: 0, Progress: 0.5, Index: 1},
	})
	tor := qbit.Torrent{Name: "Wii", Hash: hash, Progress: 0.78}
	job := map[string]interface{}{"title": "Animal Crossing.zip"}
	if !m.jobFileReady(job, tor) {
		t.Fatal("selected archive file at 100% should be ready")
	}
	job = map[string]interface{}{"title": "Other.zip"}
	if m.jobFileReady(job, tor) {
		t.Fatal("deselected/incomplete file should not be ready")
	}
}

func TestJobCompletedFields(t *testing.T) {
	fields := jobCompleted("Moved to RomM (Game Boy)")
	if fields["status"] != "completed" || fields["detail"] != "Moved to RomM (Game Boy)" {
		t.Fatalf("unexpected fields: %#v", fields)
	}
}

func TestResolveImportContentPathArchiveMember(t *testing.T) {
	cfg := newTestConfig(t)
	jobs := newTestJobs(t)
	qm := newQbitMock(t)
	cfg.QBURL = qm.srv.URL
	m := New(cfg, jobs, qm.client())

	hash := "archive-hash"
	torrent := &qbit.Torrent{
		Name:        "Game Boy",
		Hash:        hash,
		ContentPath: "/data/torrents/console-incomplete/Minerva_Myrient",
	}
	jobID := "job-1"
	jobs.Set(jobID, map[string]interface{}{
		"title":     "Trip World (Europe).zip",
		"info_hash": hash,
	})
	qm.setFiles([]qbit.TorrentFile{
		{Name: "Minerva_Myrient/No-Intro/Nintendo - Game Boy/Trip World (Europe).zip", Index: 0},
		{Name: "Minerva_Myrient/No-Intro/Nintendo - Game Boy/Other.zip", Index: 1},
	})

	got := m.resolveImportContentPath(jobID, torrent)
	want := filepath.Join(torrent.ContentPath, "No-Intro", "Nintendo - Game Boy", "Trip World (Europe).zip")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTorrentFileContentPathStripsTorrentRoot(t *testing.T) {
	root := "/data/torrents/console-incomplete/Minerva_Myrient"
	got := torrentFileContentPath(root, "Minerva_Myrient/No-Intro/Nintendo - Game Boy/Trip World (Europe).zip")
	want := filepath.Join(root, "No-Intro", "Nintendo - Game Boy", "Trip World (Europe).zip")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTorrentFileContentPathWithoutTorrentRootPrefix(t *testing.T) {
	root := "/data/torrents/console-incomplete/Minerva_Myrient"
	got := torrentFileContentPath(root, "No-Intro/Nintendo - Game Boy/Trip World (Europe).zip")
	want := filepath.Join(root, "No-Intro", "Nintendo - Game Boy", "Trip World (Europe).zip")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTorrentWantedCompleteSelectedFiles(t *testing.T) {
	cfg := newTestConfig(t)
	jobs := newTestJobs(t)
	qm := newQbitMock(t)
	cfg.QBURL = qm.srv.URL
	m := New(cfg, jobs, qm.client())

	hash := "sel-hash"
	qm.setFiles([]qbit.TorrentFile{
		{Name: "a.zip", Priority: 1, Progress: 1.0, Index: 0},
		{Name: "b.zip", Priority: 0, Progress: 0.0, Index: 1},
	})
	tor := &qbit.Torrent{Hash: hash, Progress: 0.5}
	if !m.torrentWantedComplete(tor) {
		t.Fatal("wanted file complete should report ready")
	}
	qm.setFiles([]qbit.TorrentFile{
		{Name: "a.zip", Priority: 1, Progress: 0.5, Index: 0},
		{Name: "b.zip", Priority: 0, Progress: 0.0, Index: 1},
	})
	if m.torrentWantedComplete(tor) {
		t.Fatal("incomplete wanted file should not report ready")
	}
}

func TestWatchGameTorrentImportsReadyArchiveJobWithoutSibling(t *testing.T) {
	cfg := newTestConfig(t)
	jobs := newTestJobs(t)
	qm := newQbitMock(t)
	cfg.QBURL = qm.srv.URL
	cfg.FileListScanEnabled = false
	m := New(cfg, jobs, qm.client())

	hash := "gb-multi"
	contentRoot := filepath.Join(t.TempDir(), "Minerva_Myrient")
	romDir := filepath.Join(contentRoot, "No-Intro", "Nintendo - Game Boy")
	writeFileT(t, filepath.Join(romDir, "Trip World (Europe).zip"), []byte("rom"))

	qm.setFiles([]qbit.TorrentFile{
		{Name: "Minerva_Myrient/No-Intro/Nintendo - Game Boy/Trip World (Europe).zip", Priority: 1, Progress: 1.0, Index: 0},
		{Name: "Minerva_Myrient/No-Intro/Nintendo - Game Boy/Other Game (Europe).zip", Priority: 1, Progress: 0.5, Index: 1},
	})
	tor := qbit.Torrent{Name: "Game Boy", Hash: hash, Progress: 0.75, ContentPath: contentRoot}
	qm.setTorrents([]qbit.Torrent{tor})

	jobs.Set("job-a", map[string]interface{}{
		"status": "downloading", "title": "Trip World (Europe).zip",
		"info_hash": hash, "platform": "Game Boy", "platform_slug": "gb",
	})
	jobs.Set("job-b", map[string]interface{}{
		"status": "downloading", "title": "Other Game (Europe).zip",
		"info_hash": hash, "platform": "Game Boy", "platform_slug": "gb",
	})

	go m.watchGameTorrent("job-a", hash, "Trip World (Europe).zip", "Game Boy", "gb", false)
	waitJobStatus(t, jobs, "job-a", "completed", minPollTimeout)

	jobB, ok := jobs.Get("job-b")
	if !ok {
		t.Fatal("job-b missing")
	}
	if status, _ := jobB["status"].(string); status != "downloading" {
		t.Errorf("job-b status = %q, want downloading while its file is incomplete", status)
	}
}

func TestImportReadyHashJobsSkipsNotReadySibling(t *testing.T) {
	cfg := newTestConfig(t)
	jobs := newTestJobs(t)
	qm := newQbitMock(t)
	cfg.QBURL = qm.srv.URL
	m := New(cfg, jobs, qm.client())

	hash := "ready-skip"
	contentRoot := filepath.Join(t.TempDir(), "Minerva_Myrient")
	romDir := filepath.Join(contentRoot, "No-Intro", "Nintendo - Game Boy")
	writeFileT(t, filepath.Join(romDir, "Trip World (Europe).zip"), []byte("rom"))

	qm.setFiles([]qbit.TorrentFile{
		{Name: "Minerva_Myrient/No-Intro/Nintendo - Game Boy/Trip World (Europe).zip", Priority: 1, Progress: 1.0, Index: 0},
		{Name: "Minerva_Myrient/No-Intro/Nintendo - Game Boy/Other Game (Europe).zip", Priority: 1, Progress: 0.5, Index: 1},
	})
	tor := qbit.Torrent{Name: "Game Boy", Hash: hash, Progress: 0.75, ContentPath: contentRoot}

	jobs.Set("job-a", map[string]interface{}{
		"status": "downloading", "title": "Trip World (Europe).zip",
		"info_hash": hash, "platform": "Game Boy", "platform_slug": "gb",
	})
	jobs.Set("job-b", map[string]interface{}{
		"status": "downloading", "title": "Other Game (Europe).zip",
		"info_hash": hash, "platform": "Game Boy", "platform_slug": "gb",
	})

	if done := m.importReadyHashJobs(tor, "job-a", "Game Boy", "gb", false); done {
		t.Fatal("importReadyHashJobs returned done while sibling still active")
	}
	waitJobStatus(t, jobs, "job-a", "completed", minPollTimeout)

	jobB, _ := jobs.Get("job-b")
	if status, _ := jobB["status"].(string); status != "downloading" {
		t.Errorf("job-b status = %q, want downloading", status)
	}
}

func TestResolveImportContentPathDoesNotFallBackToArchiveRoot(t *testing.T) {
	cfg := newTestConfig(t)
	jobs := newTestJobs(t)
	qm := newQbitMock(t)
	cfg.QBURL = qm.srv.URL
	m := New(cfg, jobs, qm.client())

	root := "/data/torrents/console-incomplete/Minerva_Myrient"
	torrent := &qbit.Torrent{Name: "Xbox", Hash: "no-files", ContentPath: root}
	jobs.Set("job-x", map[string]interface{}{
		"title": "Halo (USA).zip", "info_hash": "no-files", "platform_slug": "xbox",
	})
	qm.setFiles(nil)

	got := m.resolveImportContentPath("job-x", torrent)
	if got == root || filepath.Base(got) == "Minerva_Myrient" {
		t.Fatalf("fell back to archive root: %q", got)
	}
	if filepath.Base(got) != "Halo (USA).zip" {
		t.Fatalf("got %q, want a Halo (USA).zip path", got)
	}
}

func TestTorrentFileContentPathStripsGenericRootWhenRenamed(t *testing.T) {
	root := "/data/torrents/console-incomplete/Xbox"
	got := torrentFileContentPath(root, "Minerva_Myrient/Redump/Microsoft - Xbox/Halo (USA).zip")
	want := filepath.Join(root, "Redump", "Microsoft - Xbox", "Halo (USA).zip")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTorrentFileContentPathSingleFileDoesNotNest(t *testing.T) {
	file := "/data/torrents/console/Halo (USA).zip"
	got := torrentFileContentPath(file, "Halo (USA).zip")
	if got != file {
		t.Fatalf("got %q, want the file itself (not nested)", got)
	}
	got = torrentFileContentPath(file, "SomeTorrent/Halo (USA).zip")
	if got != file {
		t.Fatalf("got %q, want the file itself when qB prefixes the torrent name", got)
	}
}

func TestArchiveMemberUnderSlugPicksExistingPrefix(t *testing.T) {
	reg, err := sources.Default()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	base := "Summer Games (USA).zip"
	tape := filepath.Join(root, "No-Intro", "Commodore - Commodore 64 (Tapes)", base)
	writeFileT(t, tape, []byte("tap"))
	got := archiveMemberUnderSlug(root, base, "c64", reg)
	if got != tape {
		t.Fatalf("got %q, want tape path %q", got, tape)
	}
}

func TestRommSlugForImportRejectsArchiveFolder(t *testing.T) {
	reg, err := sources.Default()
	if err != nil {
		t.Fatal(err)
	}
	file := "/data/Minerva_Myrient/No-Intro/Nintendo - Game Boy/Trip World (Europe).zip"
	root := "/data/Minerva_Myrient"
	if got := rommSlugForImport("Minerva_Myrient", file, root, reg); got != "gb" {
		t.Fatalf("generic slug remapped to %q, want gb", got)
	}
	if got := rommSlugForImport("gb", file, root, reg); got != "gb" {
		t.Fatalf("kept slug = %q, want gb", got)
	}
	if got := rommSlugForImport("pc", file, root, reg); got != "gb" {
		t.Fatalf("watcher pc slug remapped to %q, want gb", got)
	}
}

func TestOrganizeGameArchiveMemberLandsInSlugNotTree(t *testing.T) {
	cfg := newTestConfig(t)
	reg, err := sources.Default()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Sources = reg
	jobs := newTestJobs(t)
	qm := newQbitMock(t)
	cfg.QBURL = qm.srv.URL
	m := New(cfg, jobs, qm.client())

	root := filepath.Join(t.TempDir(), "Minerva_Myrient")
	rom := filepath.Join(root, "No-Intro", "Nintendo - Game Boy", "Trip World (Europe).zip")
	writeFileT(t, rom, []byte("rom"))
	writeFileT(t, filepath.Join(root, "Redump", "Microsoft - Xbox", "Halo (USA).zip"), []byte("xbox"))

	hash := "gb-one"
	qm.setFiles([]qbit.TorrentFile{
		{Name: "Minerva_Myrient/No-Intro/Nintendo - Game Boy/Trip World (Europe).zip", Priority: 1, Progress: 1.0, Index: 0},
		{Name: "Minerva_Myrient/Redump/Microsoft - Xbox/Halo (USA).zip", Priority: 1, Progress: 1.0, Index: 1},
	})
	tor := qbit.Torrent{Name: "Game Boy", Hash: hash, Progress: 0.5, ContentPath: root}
	jobID := "job-gb"
	jobs.Set(jobID, map[string]interface{}{
		"status": "downloading", "title": "Trip World (Europe).zip",
		"info_hash": hash, "platform": "Game Boy", "platform_slug": "gb",
	})

	if m.organizeGame(jobID, &tor, "Game Boy", "gb", false, 1) {
		t.Fatal("organizeGame reported retryable")
	}
	dest := filepath.Join(cfg.GamesRomsPath, "gb", "Trip World (Europe).zip")
	if !pathExists(dest) {
		t.Fatalf("ROM not at Structure A dest %s", dest)
	}
	if pathExists(filepath.Join(cfg.GamesRomsPath, "gb", "Minerva_Myrient")) {
		t.Fatal("copied archive tree into gb")
	}
	if pathExists(filepath.Join(cfg.GamesRomsPath, "Minerva_Myrient")) {
		t.Fatal("invented Minerva_Myrient platform")
	}
	if pathExists(filepath.Join(cfg.GamesRomsPath, "xbox", "Minerva_Myrient")) ||
		pathExists(filepath.Join(cfg.GamesRomsPath, "xbox", "Halo (USA).zip")) {
		t.Fatal("imported sibling platform files into this job")
	}
	if !pathExists(dest + ".gamarr.json") {
		t.Fatal("missing .gamarr.json sidecar")
	}
	job, _ := jobs.Get(jobID)
	if status, _ := job["status"].(string); status != "completed" {
		t.Fatalf("status = %q, want completed", status)
	}
}

func TestOrganizeGameArchiveTreeFlattensBySlug(t *testing.T) {
	cfg := newTestConfig(t)
	reg, err := sources.Default()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Sources = reg
	jobs := newTestJobs(t)
	qm := newQbitMock(t)
	m := New(cfg, jobs, qm.client())

	root := filepath.Join(t.TempDir(), "Minerva_Myrient")
	writeFileT(t, filepath.Join(root, "No-Intro", "Nintendo - Game Boy", "Trip World (Europe).zip"), []byte("rom"))
	writeFileT(t, filepath.Join(root, "Redump", "Microsoft - Xbox", "Halo (USA).zip"), []byte("xbox"))

	jobID := "job-tree"
	jobs.Set(jobID, map[string]interface{}{
		"status": "organizing", "title": "Minerva Archive",
		"platform": "Unknown", "platform_slug": "", "is_pc": false,
	})
	tor := qbit.Torrent{Name: "Minerva Archive", Hash: "tree", ContentPath: root}
	if m.organizeGame(jobID, &tor, "Unknown", "", false, 1) {
		t.Fatal("flatten import should not retry")
	}
	if !pathExists(filepath.Join(cfg.GamesRomsPath, "gb", "Trip World (Europe).zip")) {
		t.Fatal("GB ROM not flattened")
	}
	if !pathExists(filepath.Join(cfg.GamesRomsPath, "xbox", "Halo (USA).zip")) {
		t.Fatal("Xbox ROM not flattened")
	}
	if pathExists(filepath.Join(cfg.GamesRomsPath, "Minerva_Myrient")) ||
		pathExists(filepath.Join(cfg.GamesVaultPath, "Minerva_Myrient")) ||
		pathExists(filepath.Join(cfg.GamesRomsPath, "pc", "Minerva_Myrient")) {
		t.Fatal("imported archive tree as a library folder")
	}
	job, _ := jobs.Get(jobID)
	if status, _ := job["status"].(string); status != "completed" {
		t.Fatalf("status = %q, want completed", status)
	}
}

func TestOrganizeGameFlattensC64PPCollection(t *testing.T) {
	cfg := newTestConfig(t)
	reg, err := sources.Default()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Sources = reg
	jobs := newTestJobs(t)
	qm := newQbitMock(t)
	m := New(cfg, jobs, qm.client())

	root := filepath.Join(t.TempDir(), "C64")
	writeFileT(t, filepath.Join(root, "No-Intro", "Commodore - Commodore 64 (PP)", "Turrican (USA).zip"), []byte("c64"))
	writeFileT(t, filepath.Join(root, "No-Intro", "Commodore - VIC-20", "Omega Race (USA).zip"), []byte("vic"))

	jobID := "job-c64"
	jobs.Set(jobID, map[string]interface{}{
		"status": "organizing", "title": "Minerva Archive",
		"platform": "C64", "platform_slug": "C64",
	})
	tor := qbit.Torrent{Name: "C64", Hash: "c64", ContentPath: root}
	if m.organizeGame(jobID, &tor, "C64", "C64", false, 1) {
		t.Fatal("c64 flatten should not retry")
	}
	if !pathExists(filepath.Join(cfg.GamesRomsPath, "c64", "Turrican (USA).zip")) {
		t.Fatal("C64 (PP) ROM not imported")
	}
	if pathExists(filepath.Join(cfg.GamesRomsPath, "vic-20", "Omega Race (USA).zip")) {
		t.Fatal("filtered slug imported a VIC-20 file")
	}
}

func TestOrganizeGameFlattensVIC20WithCardSlug(t *testing.T) {
	cfg := newTestConfig(t)
	reg, err := sources.Default()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Sources = reg
	jobs := newTestJobs(t)
	m := New(cfg, jobs, newQbitMock(t).client())

	root := filepath.Join(t.TempDir(), "Minerva_Myrient")
	writeFileT(t, filepath.Join(root, "No-Intro", "Commodore - VIC-20", "Omega Race (USA).zip"), []byte("vic"))
	writeFileT(t, filepath.Join(root, "No-Intro", "Commodore - Commodore 64 (PP)", "Turrican (USA).zip"), []byte("c64"))

	jobID := "job-vic"
	jobs.Set(jobID, map[string]interface{}{
		"status": "organizing", "title": "Unknown",
		"platform": "VIC20", "platform_slug": "vic20",
	})
	tor := qbit.Torrent{Name: "Unknown", Hash: "vic", ContentPath: root}
	if m.organizeGame(jobID, &tor, "VIC20", "vic20", false, 1) {
		t.Fatal("vic-20 flatten should not retry")
	}
	if !pathExists(filepath.Join(cfg.GamesRomsPath, "vic-20", "Omega Race (USA).zip")) {
		t.Fatal("VIC-20 ROM not imported")
	}
	if pathExists(filepath.Join(cfg.GamesRomsPath, "c64", "Turrican (USA).zip")) {
		t.Fatal("filtered slug imported a C64 file")
	}
}

func TestOrganizeGameDecodesPercentBasename(t *testing.T) {
	cfg := newTestConfig(t)
	jobs := newTestJobs(t)
	qm := newQbitMock(t)
	cfg.QBURL = qm.srv.URL
	m := New(cfg, jobs, qm.client())

	root := filepath.Join(t.TempDir(), "Minerva_Myrient")
	encoded := "Silent%20Hill%20-%20Shattered%20Memories%20%28Europe%29.zip"
	rom := filepath.Join(root, "Redump", "Nintendo - Wii - NKit RVZ [zstd-19-128k]", encoded)
	writeFileT(t, rom, []byte("rvz-bytes"))
	qm.setFiles([]qbit.TorrentFile{
		{Name: "Minerva_Myrient/Redump/Nintendo - Wii - NKit RVZ [zstd-19-128k]/" + encoded, Priority: 1, Progress: 1.0, Index: 0},
	})
	jobID := "job-enc"
	jobs.Set(jobID, map[string]interface{}{
		"title": encoded, "platform": "Wii", "platform_slug": "wii",
	})
	tor := qbit.Torrent{Name: "Wii", Hash: "enc", ContentPath: root}
	m.organizeGame(jobID, &tor, "Wii", "wii", false, 1)

	want := filepath.Join(cfg.GamesRomsPath, "wii", "Silent Hill - Shattered Memories (Europe).zip")
	if !pathExists(want) {
		t.Fatalf("decoded dest missing: %s", want)
	}
	if pathExists(filepath.Join(cfg.GamesRomsPath, "wii", encoded)) {
		t.Fatal("left URL-encoded basename in the library")
	}
}

func TestOrganizeGameRefusesHTMLBody(t *testing.T) {
	cfg := newTestConfig(t)
	jobs := newTestJobs(t)
	m := New(cfg, jobs, newQbitMock(t).client())
	html := []byte("<!DOCTYPE html>\n<html><title>Fast and Reliable Video Game Collections | Myrient</title></html>")
	src := filepath.Join(t.TempDir(), "Silent%20Hill%20(Europe).zip")
	writeFileT(t, src, html)
	jobID := "job-html"
	jobs.Set(jobID, map[string]interface{}{"title": "Silent Hill (Europe).zip", "platform_slug": "wii"})
	m.organizeGame(jobID, &qbit.Torrent{Name: "Wii", Hash: "html", ContentPath: src}, "Wii", "wii", false, 1)
	if pathExists(filepath.Join(cfg.GamesRomsPath, "wii", "Silent Hill (Europe).zip")) {
		t.Fatal("imported Myrient HTML as a ROM")
	}
	job, _ := jobs.Get(jobID)
	if status, _ := job["status"].(string); status != "error" {
		t.Fatalf("status = %q, want error", status)
	}
}

func TestOrganizeGameDoesNotOverwriteExistingROM(t *testing.T) {
	cfg := newTestConfig(t)
	jobs := newTestJobs(t)
	qm := newQbitMock(t)
	cfg.QBURL = qm.srv.URL
	m := New(cfg, jobs, qm.client())

	root := filepath.Join(t.TempDir(), "Minerva_Myrient")
	rom := filepath.Join(root, "No-Intro", "Nintendo - Game Boy", "Trip World (Europe).zip")
	writeFileT(t, rom, []byte("new"))
	dest := filepath.Join(cfg.GamesRomsPath, "gb", "Trip World (Europe).zip")
	writeFileT(t, dest, []byte("existing"))

	qm.setFiles([]qbit.TorrentFile{
		{Name: "Minerva_Myrient/No-Intro/Nintendo - Game Boy/Trip World (Europe).zip", Priority: 1, Progress: 1.0, Index: 0},
	})
	jobID := "job-dup"
	jobs.Set(jobID, map[string]interface{}{
		"title": "Trip World (Europe).zip", "platform_slug": "gb",
	})
	tor := qbit.Torrent{Name: "Game Boy", Hash: "dup", ContentPath: root}
	m.organizeGame(jobID, &tor, "Game Boy", "gb", false, 1)

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "existing" {
		t.Fatalf("overwrote dest: %q", got)
	}
}
