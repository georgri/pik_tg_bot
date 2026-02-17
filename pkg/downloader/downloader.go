package downloader

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/georgri/pik_tg_bot/pkg/flatstorage"
)

const (
	// PIK's search pages on pik.ru use this filter service for pagination ("Показать ещё").
	// It contains correct, up-to-date bulk membership (e.g. bulk 10272 inside bnab).
	PikUrl = "https://filter.dev-service.tech/api/v1/filter/flat-by-block"

	// Stable pagination: always request sorted results, otherwise pages can overlap / drift.
	// NOTE: flatLimit is not strictly honored on page=1 (server may return 20+ items),
	// but lastPage/count stay consistent and iterating pages 1..lastPage yields all flats.
	UrlParams = "type=1,2&location=2,3&sortBy=price&orderBy=asc&onlyFlats=1&flatLimit=8"

	flatPageFlag = "flatPage"

	// TODO: download this url to monitor new projects
	BlocksUrl = "https://filter.dev-service.tech/api/v1/filter/block?type=1,2&location=2,3&flatLimit=1&blockLimit=2000"

	// pik.ru sometimes "flaps" and returns the unrelated main page payload.
	// Retry fast to get the intended JSON response.
	flapMaxAttempts = 50
	flapRetryDelay  = 20 * time.Millisecond // ~50 attempts/sec
)

type HTTPResponse struct {
	URL         string
	StatusCode  int
	Status      string
	ContentType string
	Body        []byte
}

type IDCount struct {
	ID    int64
	Count int
}

// LocalFilterInfo describes why local file filtering produced no messages.
// It is intentionally string-friendly to be logged in errors.
type LocalFilterInfo struct {
	BlockID int64
	URL     string

	LastPage     int
	PagesFetched int

	DownloadedFlats       int
	DownloadedUniqueIDs   int
	DownloadedZeroIDs     int
	DownloadedDuplicateID int // number of duplicate occurrences beyond the first for each ID
	TopDuplicateIDs       []IDCount

	StorageFile     string
	StorageExists   bool
	StorageModTime  string
	StoredFlats     int
	StoredUniqueIDs int
	StoredZeroIDs   int

	OverlapUniqueIDs int
	NewUniqueIDs     int

	NewFlatsAfterIDFilter int
	ReturnedMessages      int
}

func (i *LocalFilterInfo) String() string {
	if i == nil {
		return "<nil>"
	}

	parts := []string{
		fmt.Sprintf("downloaded=%d flats (uniqueIDs=%d, zeroIDs=%d, dupOccur=%d)",
			i.DownloadedFlats, i.DownloadedUniqueIDs, i.DownloadedZeroIDs, i.DownloadedDuplicateID),
		fmt.Sprintf("pagesFetched=%d/%d", i.PagesFetched, i.LastPage),
		fmt.Sprintf("storage=%q (exists=%t, modTime=%s, stored=%d flats, uniqueIDs=%d, zeroIDs=%d)",
			i.StorageFile, i.StorageExists, i.StorageModTime, i.StoredFlats, i.StoredUniqueIDs, i.StoredZeroIDs),
		fmt.Sprintf("overlapUniqueIDs=%d, newUniqueIDs=%d", i.OverlapUniqueIDs, i.NewUniqueIDs),
		fmt.Sprintf("newFlatsAfterIDFilter=%d, returnedMessages=%d", i.NewFlatsAfterIDFilter, i.ReturnedMessages),
	}

	if len(i.TopDuplicateIDs) > 0 {
		dupParts := make([]string, 0, len(i.TopDuplicateIDs))
		for _, dc := range i.TopDuplicateIDs {
			dupParts = append(dupParts, fmt.Sprintf("%d×%d", dc.ID, dc.Count))
		}
		parts = append(parts, fmt.Sprintf("topDupIDs=[%s]", strings.Join(dupParts, ", ")))
	}

	return strings.Join(parts, "; ")
}

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func addPikBrowserLikeHeaders(req *http.Request) {
	if req == nil || req.URL == nil {
		return
	}

	// Add browser-like headers only for PIK domains to reduce "flapping"
	// (sometimes returning the unrelated main page HTML instead of JSON).
	host := strings.ToLower(req.URL.Hostname())
	if !strings.Contains(host, "pik-service.ru") && !strings.Contains(host, "pik.ru") && !strings.Contains(host, "dev-service.tech") {
		return
	}

	// Do NOT set Accept-Encoding: Go's http.Transport will add "gzip" and
	// transparently decode it. Advertising "br"/"zstd" would risk receiving
	// an encoding we can't decode.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 YaBrowser/25.10.0.0 Safari/537.36")
	// For API endpoints, prefer JSON accept to avoid content negotiation surprises.
	if strings.Contains(req.URL.Path, "/api/") {
		req.Header.Set("Accept", "application/json, text/plain, */*")
	} else {
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	}
	req.Header.Set("Accept-Language", "ru,en;q=0.9")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	// These are often used by bot-detection heuristics; harmless for an API endpoint.
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("sec-ch-ua", "\"Chromium\";v=\"140\", \"Not=A?Brand\";v=\"24\", \"YaBrowser\";v=\"25.10\", \"Yowser\";v=\"2.5\"")
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", "\"macOS\"")
}

func GetURLResponse(url string) (*HTTPResponse, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request for %s: %w", url, err)
	}
	addPikBrowserLikeHeaders(req)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, &NetworkError{URL: url, Err: err}
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed reading response body from %s: %w", url, readErr)
	}

	contentType := resp.Header.Get("Content-Type")
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, &HTTPStatusError{
			URL:         url,
			StatusCode:  resp.StatusCode,
			Status:      resp.Status,
			ContentType: contentType,
			BodySnippet: snippet(body, 300),
		}
	}

	return &HTTPResponse{
		URL:         url,
		StatusCode:  resp.StatusCode,
		Status:      resp.Status,
		ContentType: contentType,
		Body:        body,
	}, nil
}

type flapCheckBody struct {
	Success bool `json:"success"`
	Data    struct {
		Stats struct {
			Blocks      []int64 `json:"blocks"`
			CountBlocks *int    `json:"countBlocks"`
		} `json:"stats"`
	} `json:"data"`
}

func containsInt64(vs []int64, x int64) bool {
	for _, v := range vs {
		if v == x {
			return true
		}
	}
	return false
}

func isHTMLLike(meta *HTTPResponse) bool {
	if meta == nil {
		return false
	}
	ct := strings.ToLower(meta.ContentType)
	if strings.Contains(ct, "text/html") {
		return true
	}
	trimmed := strings.TrimSpace(string(meta.Body))
	if trimmed == "" {
		return false
	}
	if trimmed[0] == '<' {
		return true
	}
	// Some responses are HTML but served as text/plain.
	l := strings.ToLower(trimmed)
	return strings.Contains(l, "<html") || strings.Contains(l, "<!doctype")
}

func isFlapForBlock(meta *HTTPResponse, expectedBlockID int64) (bool, string) {
	if meta == nil || expectedBlockID == 0 {
		return false, ""
	}
	if isHTMLLike(meta) {
		return true, "html-main-page"
	}

	trimmed := strings.TrimSpace(string(meta.Body))
	if trimmed == "" {
		return true, "empty-body"
	}
	// If it's not even JSON-looking, treat as flap/main-page-ish.
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return true, "non-json-body"
	}

	var chk flapCheckBody
	if err := json.Unmarshal(meta.Body, &chk); err != nil {
		// Not a flap by definition: let the JSON unmarshal error be reported upstream,
		// unless it was clearly HTML already (handled above).
		return false, ""
	}

	// If API explicitly says multiple blocks or returns a different block list => flap.
	if chk.Data.Stats.CountBlocks != nil && *chk.Data.Stats.CountBlocks > 1 {
		return true, fmt.Sprintf("countBlocks=%d", *chk.Data.Stats.CountBlocks)
	}
	if len(chk.Data.Stats.Blocks) > 0 && !containsInt64(chk.Data.Stats.Blocks, expectedBlockID) {
		return true, "stats.blocks-mismatch"
	}

	return false, ""
}

func GetURLResponseWithFlapRetries(url string, expectedBlockID int64) (*HTTPResponse, error) {
	var last *HTTPResponse
	for attempt := 1; attempt <= flapMaxAttempts; attempt++ {
		meta, err := GetURLResponse(url)
		if err != nil {
			return nil, err
		}
		last = meta

		if expectedBlockID != 0 {
			flap, reason := isFlapForBlock(meta, expectedBlockID)
			if flap {
				if attempt < flapMaxAttempts {
					time.Sleep(flapRetryDelay)
					continue
				}
				return nil, &FlapError{
					URL:             url,
					ExpectedBlockID: expectedBlockID,
					Attempts:        attempt,
					ContentType:     meta.ContentType,
					Reason:          reason,
					BodySnippet:     snippet(meta.Body, 300),
				}
			}
		}

		return meta, nil
	}

	// Should never reach here due to loop bounds, but keep a safe fallback.
	if last == nil {
		return nil, &FlapError{
			URL:             url,
			ExpectedBlockID: expectedBlockID,
			Attempts:        flapMaxAttempts,
			Reason:          "no-attempts",
		}
	}
	return nil, &FlapError{
		URL:             url,
		ExpectedBlockID: expectedBlockID,
		Attempts:        flapMaxAttempts,
		ContentType:     last.ContentType,
		Reason:          "exhausted",
		BodySnippet:     snippet(last.Body, 300),
	}
}

func GetUrl(url string) ([]byte, error) {
	meta, err := GetURLResponse(url)
	if err != nil {
		return nil, err
	}
	return meta.Body, nil
}

func GetFlatsSinglePage(url string, expectedBlockID int64) (*flatstorage.MessageData, error) {
	meta, err := GetURLResponseWithFlapRetries(url, expectedBlockID)
	if err != nil {
		return nil, err
	}

	msgData, err := flatstorage.UnmarshallFlats(meta.Body)
	if err != nil {
		return nil, &ResponseUnmarshalError{
			URL:         url,
			ContentType: meta.ContentType,
			BodySnippet: snippet(meta.Body, 300),
			Err:         err,
		}
	}

	return msgData, nil
}

func summarizeFlatIDs(flats []flatstorage.Flat) (uniqueIDs map[int64]int, zeroIDs int, duplicateOccurrences int, topDup []IDCount) {
	uniqueIDs = make(map[int64]int, len(flats))
	for _, f := range flats {
		id := f.ID
		uniqueIDs[id]++
		if id == 0 {
			zeroIDs++
		}
	}

	for _, c := range uniqueIDs {
		if c > 1 {
			duplicateOccurrences += c - 1
		}
	}

	dups := make([]IDCount, 0)
	for id, c := range uniqueIDs {
		if c > 1 {
			dups = append(dups, IDCount{ID: id, Count: c})
		}
	}

	sort.Slice(dups, func(a, b int) bool {
		if dups[a].Count == dups[b].Count {
			return dups[a].ID < dups[b].ID
		}
		return dups[a].Count > dups[b].Count
	})
	if len(dups) > 5 {
		dups = dups[:5]
	}
	return uniqueIDs, zeroIDs, duplicateOccurrences, dups
}

func GetFlats(blockID int64) (messages []string, updateCallback func() error, info *LocalFilterInfo, err error) {
	u, err := url.Parse(fmt.Sprintf("%v/%v", PikUrl, blockID))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to build flats url: %w", err)
	}
	q := u.Query()
	// Keep these params in sync with `UrlParams` constant (used for debug/logging and legacy callers).
	q.Set("type", "1,2")
	q.Set("location", "2,3")
	q.Set("sortBy", "price")
	q.Set("orderBy", "asc")
	q.Set("onlyFlats", "1")
	q.Set("flatLimit", "8")
	q.Set(flatPageFlag, "1")
	u.RawQuery = q.Encode()
	flatsURL := u.String()

	info = &LocalFilterInfo{
		BlockID:        blockID,
		URL:            flatsURL,
		PagesFetched:   0,
		StorageModTime: "",
	}

	msgData, err := GetFlatsSinglePage(flatsURL, blockID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to fetch flats page 1: %w", err)
	}
	info.PagesFetched = 1

	if msgData.LastPage > 1 {
		for i := 2; i <= msgData.LastPage; i++ {
			addU, parseErr := url.Parse(flatsURL)
			if parseErr != nil {
				return nil, nil, nil, fmt.Errorf("failed to parse flats url for page %d: %w", i, parseErr)
			}
			addQ := addU.Query()
			addQ.Set(flatPageFlag, fmt.Sprintf("%d", i))
			addU.RawQuery = addQ.Encode()
			addUrl := addU.String()
			addMsgData, err := GetFlatsSinglePage(addUrl, blockID)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to fetch flats page %d: %w", i, err)
			}
			msgData.Flats = append(msgData.Flats, addMsgData.Flats...)
			info.PagesFetched++
		}
	}

	if len(msgData.Flats) == 0 {
		// If the response was parsed successfully but has no flats, surface the URL and a small snippet
		// to help distinguish "empty response" from "network/HTTP/parsing" issues.
		meta, metaErr := GetURLResponse(flatsURL)
		if metaErr != nil {
			// Prefer the meta error (network / status) while still allowing errors.Is(..., ErrorZeroFlats).
			return nil, nil, nil, fmt.Errorf("%w; additionally failed to re-fetch response meta: %v", ErrorZeroFlats, metaErr)
		}
		return nil, nil, nil, &ZeroFlatsError{
			URL:         flatsURL,
			StatusCode:  meta.StatusCode,
			Status:      meta.Status,
			ContentType: meta.ContentType,
			LastPage:    msgData.LastPage,
			BodySnippet: snippet(meta.Body, 300),
		}
	}

	msgData.CalcAveragePrices()

	origMsgData := msgData.Copy()

	info.LastPage = msgData.LastPage
	info.DownloadedFlats = len(origMsgData.Flats)
	downloadedIDs, downloadedZero, downloadedDupOccur, topDup := summarizeFlatIDs(origMsgData.Flats)
	info.DownloadedUniqueIDs = len(downloadedIDs)
	info.DownloadedZeroIDs = downloadedZero
	info.DownloadedDuplicateID = downloadedDupOccur
	info.TopDuplicateIDs = topDup

	info.StorageFile = flatstorage.GetStorageFileName(origMsgData)
	if info.StorageFile != "" {
		if st, statErr := os.Stat(info.StorageFile); statErr == nil {
			info.StorageExists = true
			info.StorageModTime = st.ModTime().Format(time.RFC3339)
		}

		if oldMsg, readErr := flatstorage.ReadFlatStorage(info.StorageFile); readErr == nil && oldMsg != nil {
			info.StoredFlats = len(oldMsg.Flats)
			oldIDs, oldZero, _, _ := summarizeFlatIDs(oldMsg.Flats)
			info.StoredUniqueIDs = len(oldIDs)
			info.StoredZeroIDs = oldZero

			overlap := 0
			for id := range downloadedIDs {
				if _, ok := oldIDs[id]; ok {
					overlap++
				}
			}
			info.OverlapUniqueIDs = overlap
			info.NewUniqueIDs = len(downloadedIDs) - overlap
		}
	}

	// filter through local file (MVP)
	var res []string
	res, err = flatstorage.FilterWithFlatStorage(msgData)
	if err != nil {
		return nil, nil, info, fmt.Errorf("err while reading/updating local Flats file: %v", err)
	}

	updateCallback = func() error {
		_, err = flatstorage.UpdateFlatStorage(origMsgData)
		return err
	}

	info.NewFlatsAfterIDFilter = len(msgData.Flats)
	info.ReturnedMessages = len(res)

	return res, updateCallback, info, nil
}
