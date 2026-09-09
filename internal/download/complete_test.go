package download

import (
	"path/filepath"
	"testing"

	"gamarr/internal/qbit"
)

func TestTorrentDownloadCompleteIgnoresSkippedFiles(t *testing.T) {
	files := []qbit.TorrentFile{
		{Name: "Minerva_Myrient/Wanted.zip", Size: 100, Progress: 1.0, Priority: 1, Index: 0},
		{Name: "Minerva_Myrient/Skip.zip", Size: 900, Progress: 0, Priority: 0, Index: 1},
	}
	tor := qbit.Torrent{Progress: 0.1, State: "stalledDL"}
	if !TorrentDownloadComplete(tor, files) {
		t.Fatal("wanted file is done; skipped siblings must not keep the torrent incomplete")
	}
	files[0].Progress = 0.4
	if TorrentDownloadComplete(tor, files) {
		t.Fatal("wanted file still downloading, must not be complete")
	}
}

func TestJobDownloadCompleteUsesMatchedFiles(t *testing.T) {
	files := []qbit.TorrentFile{
		{Name: "Wii/Super Paper Mario (USA).zip", Size: 100, Progress: 1.0, Priority: 1, Index: 0},
		{Name: "Wii/Mario Galaxy (USA).zip", Size: 300, Progress: 0.2, Priority: 1, Index: 1},
	}
	tor := qbit.Torrent{Progress: 0.4, State: "downloading"}
	if !JobDownloadComplete(tor, files, "Super Paper Mario (USA).zip") {
		t.Fatal("the job's own file is done")
	}
	if JobDownloadComplete(tor, files, "Mario Galaxy (USA).zip") {
		t.Fatal("sibling still downloading")
	}
	if JobDownloadComplete(tor, files, "Some FitGirl Repack") {
		t.Fatal("unrelated title must fall back to the torrent, which is not done")
	}
}

func TestWantedProgressSkipsPriorityZero(t *testing.T) {
	files := []qbit.TorrentFile{
		{Name: "a.zip", Size: 100, Progress: 0.5, Priority: 1},
		{Name: "b.zip", Size: 200, Progress: 0.1, Priority: 0},
		{Name: "c.zip", Size: 300, Progress: 0.25, Priority: 1},
	}
	got, ok := WantedProgress(files)
	if !ok {
		t.Fatal("wanted progress reported empty")
	}
	// (50 + 75) / 400 = 0.3125 → 31.2
	if got != 31.2 {
		t.Fatalf("progress = %v, want 31.2", got)
	}
}

func TestJobSourcePathsStripsTorrentRoot(t *testing.T) {
	files := []qbit.TorrentFile{
		{Name: "Minerva_Myrient/Wii/Super Paper Mario (USA).zip", Index: 0},
		{Name: "Minerva_Myrient/Wii/Other.zip", Index: 1},
	}
	got := jobSourcePaths("/data/incoming/Minerva_Myrient", files, "Super Paper Mario (USA).zip")
	want := filepath.Join("/data/incoming/Minerva_Myrient", "Wii", "Super Paper Mario (USA).zip")
	if len(got) != 1 || got[0] != want {
		t.Fatalf("paths = %v, want [%s]", got, want)
	}
}
