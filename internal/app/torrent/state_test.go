package torrent

import (
	"path/filepath"
	"testing"
	"time"
)

func TestMarkIncompleteClearsStoredCompletion(t *testing.T) {
	sm := NewStateManager(filepath.Join(t.TempDir(), "torrent-state.json"))
	t.Cleanup(sm.Stop)

	sm.Add(&storedTorrent{
		InfoHash: "test-hash",
		Name:     "test-torrent",
		AddedAt:  time.Now(),
	})
	if !sm.MarkCompleted("test-hash") {
		t.Fatal("expected torrent to be marked complete")
	}
	if !sm.MarkIncomplete("test-hash") {
		t.Fatal("expected stale completion to be cleared")
	}

	stored, ok := sm.Get("test-hash")
	if !ok {
		t.Fatal("stored torrent disappeared")
	}
	if stored.CompletedAt != nil {
		t.Fatalf("expected completedAt to be nil, got %v", stored.CompletedAt)
	}
	if sm.MarkIncomplete("test-hash") {
		t.Fatal("clearing an already empty completion should report no change")
	}
}

func TestApplyRecheckStatus(t *testing.T) {
	s := &Service{rechecks: map[string]*recheckJob{
		"checking": {},
		"failed":   {err: "hash read failed"},
	}}

	checking := TorrentStatus{InfoHash: "checking", Status: StatusPaused, DownloadSpeed: 10, UploadSpeed: 20, Eta: 30}
	s.applyRecheckStatus(&checking)
	if checking.Status != StatusChecking || checking.DownloadSpeed != 0 || checking.UploadSpeed != 0 || checking.Eta != 0 {
		t.Fatalf("unexpected checking status: %+v", checking)
	}

	failed := TorrentStatus{InfoHash: "failed", Status: StatusPaused}
	s.applyRecheckStatus(&failed)
	if failed.Status != StatusError || failed.Error == "" {
		t.Fatalf("unexpected failed status: %+v", failed)
	}
}
