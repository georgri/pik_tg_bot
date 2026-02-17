package downloader

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/georgri/pik_tg_bot/pkg/flatstorage"
	"github.com/georgri/pik_tg_bot/pkg/util"
)

// This is an integration test that hits the live PIK filter API.
// It is skipped unless PIK_LIVE=1 is set in the environment.
func TestGetFlats_Live_FetchesAllAndStores(t *testing.T) {
	if os.Getenv("PIK_LIVE") != "1" {
		t.Skip("set PIK_LIVE=1 to run live PIK API integration test")
	}
	if testing.Short() {
		t.Skip("skipping live test in -short")
	}

	// Isolate storage writes to a temp directory.
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	if err := os.MkdirAll("data", 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}

	// Make the storage filename deterministic.
	oldEnv := util.RootEnvType
	util.RootEnvType = "test"
	t.Cleanup(func() { util.RootEnvType = oldEnv })

	const blockID = int64(2214) // bnab

	// Fetch page 1 directly and read authoritative stats.count.
	page1URL := PikUrl + "/2214?" + UrlParams + "&flatPage=1"
	meta, err := GetURLResponseWithFlapRetries(page1URL, blockID)
	if err != nil {
		t.Fatalf("fetch page1 meta: %v", err)
	}
	var stats struct {
		Data struct {
			Stats struct {
				Count    int `json:"count"`
				LastPage int `json:"lastPage"`
			} `json:"stats"`
		} `json:"data"`
	}
	if err := json.Unmarshal(meta.Body, &stats); err != nil {
		t.Fatalf("unmarshal stats: %v", err)
	}
	if stats.Data.Stats.Count <= 0 {
		t.Fatalf("unexpected stats.count=%d", stats.Data.Stats.Count)
	}

	msgs, updateCallback, info, err := GetFlats(blockID)
	if err != nil {
		t.Fatalf("GetFlats(%d): %v", blockID, err)
	}
	if updateCallback == nil {
		t.Fatalf("expected non-nil updateCallback")
	}
	if info == nil {
		t.Fatalf("expected non-nil info")
	}
	if len(msgs) == 0 {
		t.Fatalf("expected some messages (got 0)")
	}

	// Prove we fetched all pages and got a complete, de-duplicated set.
	if info.PagesFetched != info.LastPage {
		t.Fatalf("expected pagesFetched==lastPage, got %d vs %d", info.PagesFetched, info.LastPage)
	}
	if info.DownloadedUniqueIDs != info.DownloadedFlats {
		t.Fatalf("expected no duplicates, flats=%d unique=%d", info.DownloadedFlats, info.DownloadedUniqueIDs)
	}
	if info.DownloadedUniqueIDs != stats.Data.Stats.Count {
		t.Fatalf("expected to fetch exactly stats.count=%d unique flats, got %d", stats.Data.Stats.Count, info.DownloadedUniqueIDs)
	}

	// Store and validate we can read the same set back.
	if err := updateCallback(); err != nil {
		t.Fatalf("updateCallback: %v", err)
	}
	if info.StorageFile == "" {
		t.Fatalf("expected info.StorageFile to be set")
	}
	if !filepath.IsAbs(info.StorageFile) {
		// storage file paths are relative to CWD; that's expected here
		if _, err := os.Stat(info.StorageFile); err != nil {
			t.Fatalf("stat storage file %q: %v", info.StorageFile, err)
		}
	}

	stored, err := flatstorage.ReadFlatStorage(info.StorageFile)
	if err != nil {
		t.Fatalf("ReadFlatStorage(%q): %v", info.StorageFile, err)
	}
	if stored == nil {
		t.Fatalf("expected non-nil stored msg")
	}

	seen := make(map[int64]struct{}, len(stored.Flats))
	for _, f := range stored.Flats {
		if f.ID == 0 {
			t.Fatalf("stored flat with id=0")
		}
		seen[f.ID] = struct{}{}
	}
	if len(seen) != stats.Data.Stats.Count {
		t.Fatalf("expected stored unique ids == stats.count=%d, got %d (stored flats=%d)",
			stats.Data.Stats.Count, len(seen), len(stored.Flats))
	}
}

