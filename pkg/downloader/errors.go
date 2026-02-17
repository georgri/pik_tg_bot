package downloader

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorZeroFlats is kept for backward compatibility with older code paths.
// Prefer using errors.Is(err, ErrorZeroFlats) rather than direct equality checks.
var ErrorZeroFlats = errors.New("got zero flats")

type NetworkError struct {
	URL string
	Err error
}

func (e *NetworkError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("network error while GET %s: %v", e.URL, e.Err)
}

func (e *NetworkError) Unwrap() error { return e.Err }

type HTTPStatusError struct {
	URL         string
	StatusCode  int
	Status      string
	ContentType string
	BodySnippet string
}

func (e *HTTPStatusError) Error() string {
	if e == nil {
		return "<nil>"
	}
	parts := []string{
		fmt.Sprintf("unexpected HTTP status while GET %s: %s", e.URL, e.Status),
	}
	if e.ContentType != "" {
		parts = append(parts, fmt.Sprintf("content-type=%s", e.ContentType))
	}
	if e.BodySnippet != "" {
		parts = append(parts, fmt.Sprintf("body=%q", e.BodySnippet))
	}
	return strings.Join(parts, "; ")
}

type ResponseUnmarshalError struct {
	URL         string
	ContentType string
	BodySnippet string
	Err         error
}

func (e *ResponseUnmarshalError) Error() string {
	if e == nil {
		return "<nil>"
	}
	parts := []string{
		fmt.Sprintf("failed to parse response JSON from %s", e.URL),
	}
	if e.ContentType != "" {
		parts = append(parts, fmt.Sprintf("content-type=%s", e.ContentType))
	}
	if e.BodySnippet != "" {
		parts = append(parts, fmt.Sprintf("body=%q", e.BodySnippet))
	}
	if e.Err != nil {
		parts = append(parts, fmt.Sprintf("err=%v", e.Err))
	}
	return strings.Join(parts, "; ")
}

func (e *ResponseUnmarshalError) Unwrap() error { return e.Err }

type ZeroFlatsError struct {
	URL         string
	StatusCode  int
	Status      string
	ContentType string
	LastPage    int
	BodySnippet string
}

func (e *ZeroFlatsError) Error() string {
	if e == nil {
		return "<nil>"
	}
	parts := []string{
		fmt.Sprintf("got zero flats from %s", e.URL),
	}
	if e.Status != "" {
		parts = append(parts, fmt.Sprintf("status=%s", e.Status))
	}
	if e.ContentType != "" {
		parts = append(parts, fmt.Sprintf("content-type=%s", e.ContentType))
	}
	if e.LastPage != 0 {
		parts = append(parts, fmt.Sprintf("lastPage=%d", e.LastPage))
	}
	if e.BodySnippet != "" {
		parts = append(parts, fmt.Sprintf("body=%q", e.BodySnippet))
	}
	return strings.Join(parts, "; ")
}

func (e *ZeroFlatsError) Unwrap() error { return ErrorZeroFlats }

// FlapError indicates a "flap" response: the PIK API returned unrelated main-page HTML
// or a JSON payload that doesn't correspond to the requested block.
type FlapError struct {
	URL             string
	ExpectedBlockID int64
	Attempts        int
	ContentType     string
	Reason          string
	BodySnippet     string
}

func (e *FlapError) Error() string {
	if e == nil {
		return "<nil>"
	}
	parts := []string{
		fmt.Sprintf("PIK API flap while GET %s", e.URL),
	}
	if e.ExpectedBlockID != 0 {
		parts = append(parts, fmt.Sprintf("expectedBlockID=%d", e.ExpectedBlockID))
	}
	if e.Attempts != 0 {
		parts = append(parts, fmt.Sprintf("attempts=%d", e.Attempts))
	}
	if e.ContentType != "" {
		parts = append(parts, fmt.Sprintf("content-type=%s", e.ContentType))
	}
	if e.Reason != "" {
		parts = append(parts, fmt.Sprintf("reason=%s", e.Reason))
	}
	if e.BodySnippet != "" {
		parts = append(parts, fmt.Sprintf("body=%q", e.BodySnippet))
	}
	return strings.Join(parts, "; ")
}

func snippet(body []byte, maxLen int) string {
	if maxLen <= 0 || len(body) == 0 {
		return ""
	}
	s := strings.TrimSpace(string(body))
	if s == "" {
		return ""
	}
	// Reduce log noise from pretty JSON / HTML.
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxLen {
		return s[:maxLen] + "…"
	}
	return s
}
