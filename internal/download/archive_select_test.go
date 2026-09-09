package download

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gamarr/internal/qbit"
)

func TestMatchTorrentFileIndexes(t *testing.T) {
	files := []qbit.TorrentFile{
		{Name: "No-Intro/Wii/Other Game (USA).zip", Size: 1, Index: 0},
		{Name: "No-Intro/Wii/Super Paper Mario (Europe, Australia) (En,Fr,De,Es,It).zip", Size: 2, Index: 1},
		{Name: "No-Intro/Wii/Super Mario Galaxy (USA).zip", Size: 3, Index: 2},
	}
	got := matchTorrentFileIndexes(files, "Super Paper Mario (Europe, Australia) (En,Fr,De,Es,It).zip")
	if len(got) != 1 || got[0] != 1 {
		t.Fatalf("got %v, want [1]", got)
	}
	got = matchTorrentFileIndexes(files, "Super Paper Mario (Europe, Australia) (En,Fr,De,Es,It)")
	if len(got) != 1 || got[0] != 1 {
		t.Fatalf("stem match got %v, want [1]", got)
	}
	if got := matchTorrentFileIndexes(files, "Missing Title.zip"); len(got) != 0 {
		t.Fatalf("got %v, want none", got)
	}
}

// A Minerva magnet is one torrent of many ROMs. qBittorrent's torrent-level
// progress is often downloaded/total_size, so a single selected file never
// pushes it to 1.0. The job still has to complete and import that file.
func TestWatchCompletesSelectedFileWhenTorrentProgressIsPartial(t *testing.T) {
	setImportRetries(t, 3, 5*time.Millisecond)
	cfg := newTestConfig(t)
	cfg.QBURL = "configured"
	jobs := newTestJobs(t)

	content := filepath.Join(t.TempDir(), "Minerva_Myrient")
	rom := filepath.Join(content, "Super Paper Mario (USA).zip")
	writeFileT(t, rom, []byte("paper-mario"))
	writeFileT(t, filepath.Join(content, "Other Game (USA).zip.!qB"), []byte(""))

	qm := newQbitMock(t)
	qm.setFiles([]qbit.TorrentFile{
		{Name: "Minerva_Myrient/Super Paper Mario (USA).zip", Size: 100, Progress: 1.0, Priority: 1, Index: 0},
		{Name: "Minerva_Myrient/Other Game (USA).zip", Size: 900, Progress: 0, Priority: 0, Index: 1},
	})
	qm.setTorrents([]qbit.Torrent{{
		Name: "Minerva_Myrient", Hash: "archive-hash", Progress: 0.1,
		State: "stalledDL", ContentPath: content,
	}})

	m := New(cfg, jobs, qm.client())
	jobID, err := m.DownloadTorrent("magnet:x", "archive-hash", "Super Paper Mario (USA).zip", "Wii", "wii", false, false)
	if err != nil {
		t.Fatalf("DownloadTorrent: %v", err)
	}
	waitJobStatus(t, jobs, jobID, "completed", minPollTimeout)

	want := filepath.Join(cfg.GamesRomsPath, "wii", "Super Paper Mario (USA).zip")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("selected ROM not imported: %v", err)
	}
	if pathExists(filepath.Join(cfg.GamesRomsPath, "wii", "Minerva_Myrient")) {
		t.Error("imported the whole archive folder instead of the selected ROM")
	}
}

func TestWatchCompletesOneArchiveJobWithoutWaitingForASibling(t *testing.T) {
	setImportRetries(t, 3, 5*time.Millisecond)
	cfg := newTestConfig(t)
	cfg.QBURL = "configured"
	jobs := newTestJobs(t)

	content := filepath.Join(t.TempDir(), "Minerva_Myrient")
	writeFileT(t, filepath.Join(content, "Super Paper Mario (USA).zip"), []byte("paper-mario"))
	writeFileT(t, filepath.Join(content, "Mario Galaxy (USA).zip"), []byte("partial"))

	qm := newQbitMock(t)
	qm.setFiles([]qbit.TorrentFile{
		{Name: "Minerva_Myrient/Super Paper Mario (USA).zip", Size: 100, Progress: 1.0, Priority: 1, Index: 0},
		{Name: "Minerva_Myrient/Mario Galaxy (USA).zip", Size: 300, Progress: 0.2, Priority: 1, Index: 1},
	})
	qm.setTorrents([]qbit.Torrent{{
		Name: "Minerva_Myrient", Hash: "shared-hash", Progress: 0.4,
		State: "downloading", ContentPath: content,
	}})

	m := New(cfg, jobs, qm.client())
	jobA, err := m.DownloadTorrent("magnet:x", "shared-hash", "Super Paper Mario (USA).zip", "Wii", "wii", false, false)
	if err != nil {
		t.Fatalf("job A: %v", err)
	}
	jobB, err := m.DownloadTorrent("magnet:x", "shared-hash", "Mario Galaxy (USA).zip", "Wii", "wii", false, false)
	if err != nil {
		t.Fatalf("job B: %v", err)
	}

	waitJobStatus(t, jobs, jobA, "completed", minPollTimeout)
	if job, ok := jobs.Get(jobB); !ok {
		t.Fatal("job B missing")
	} else if status, _ := job["status"].(string); status == "completed" {
		t.Fatal("job B completed while its file was still at 20%")
	}
	if len(qm.deletedHashes()) != 0 {
		t.Fatalf("shared torrent was removed while job B still needs it: %v", qm.deletedHashes())
	}

	qm.setFiles([]qbit.TorrentFile{
		{Name: "Minerva_Myrient/Super Paper Mario (USA).zip", Size: 100, Progress: 1.0, Priority: 1, Index: 0},
		{Name: "Minerva_Myrient/Mario Galaxy (USA).zip", Size: 300, Progress: 1.0, Priority: 1, Index: 1},
	})
	waitJobStatus(t, jobs, jobB, "completed", minPollTimeout)
}

func TestOrganizeTorrentAcceptsWantedFilesWithoutTorrentProgress(t *testing.T) {
	cfg := newTestConfig(t)
	jobs := newTestJobs(t)
	content := filepath.Join(t.TempDir(), "Cool.Game-FitGirl")
	writeFileT(t, filepath.Join(content, "setup.exe"), []byte("installer"))

	qm := newQbitMock(t)
	qm.setFiles([]qbit.TorrentFile{
		{Name: "Cool.Game-FitGirl/setup.exe", Size: 10, Progress: 1.0, Priority: 1, Index: 0},
		{Name: "Cool.Game-FitGirl/bonus.iso", Size: 90, Progress: 0, Priority: 0, Index: 1},
	})
	qm.setTorrents([]qbit.Torrent{{
		Name: "Cool.Game-FitGirl", Hash: "h-partial", Progress: 0.1, ContentPath: content,
	}})
	m := New(cfg, jobs, qm.client())

	jobID, err := m.OrganizeTorrent("h-partial", "PC", "", true)
	if err != nil {
		t.Fatalf("OrganizeTorrent: %v", err)
	}
	waitJobStatus(t, jobs, jobID, "completed", minPollTimeout)
}
