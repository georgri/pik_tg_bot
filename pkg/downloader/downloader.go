package downloader

import (
	"encoding/json"
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/flatstorage"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	PikUrl    = "https://flat.pik-service.ru/api/v1/filter/flat-by-block"
	UrlParams = "sortBy=price&orderBy=asc&onlyFlats=1&flatLimit=16"

	flatPageFlag = "flatPage"

	// TODO: download this url to monitor new projects
	BlocksUrl = "https://flat.pik-service.ru/api/v1/filter/block?type=1,2&location=2,3&flatLimit=50&blockLimit=1000&geoBox=55.33638001424489,56.14056105282492-36.96336293218961,38.11418080328337"

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

func GetURLResponse(url string) (*HTTPResponse, error) {
	resp, err := http.Get(url)
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

func GetFlats(blockID int64) (messages []string, updateCallback func() error, err error) {
	url := fmt.Sprintf("%v/%v?%v", PikUrl, blockID, UrlParams)

	msgData, err := GetFlatsSinglePage(url, blockID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch flats page 1: %w", err)
	}

	if msgData.LastPage > 1 {
		for i := 2; i <= msgData.LastPage; i++ {
			addUrl := fmt.Sprintf("%v&%v=%v", url, flatPageFlag, i)
			addMsgData, err := GetFlatsSinglePage(addUrl, blockID)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to fetch flats page %d: %w", i, err)
			}
			msgData.Flats = append(msgData.Flats, addMsgData.Flats...)
		}
	}

	if len(msgData.Flats) == 0 {
		// If the response was parsed successfully but has no flats, surface the URL and a small snippet
		// to help distinguish "empty response" from "network/HTTP/parsing" issues.
		meta, metaErr := GetURLResponse(url)
		if metaErr != nil {
			// Prefer the meta error (network / status) while still allowing errors.Is(..., ErrorZeroFlats).
			return nil, nil, fmt.Errorf("%w; additionally failed to re-fetch response meta: %v", ErrorZeroFlats, metaErr)
		}
		return nil, nil, &ZeroFlatsError{
			URL:         url,
			StatusCode:  meta.StatusCode,
			Status:      meta.Status,
			ContentType: meta.ContentType,
			LastPage:    msgData.LastPage,
			BodySnippet: snippet(meta.Body, 300),
		}
	}

	msgData.CalcAveragePrices()

	origMsgData := msgData.Copy()

	// filter through local file (MVP)
	var res []string
	res, err = flatstorage.FilterWithFlatStorage(msgData)
	if err != nil {
		return nil, nil, fmt.Errorf("err while reading/updating local Flats file: %v", err)
	}

	updateCallback = func() error {
		_, err = flatstorage.UpdateFlatStorage(origMsgData)
		return err
	}

	return res, updateCallback, nil
}
