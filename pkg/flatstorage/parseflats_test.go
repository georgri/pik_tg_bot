package flatstorage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFlat_GetPriceHistory_FiltersBefore2023AndInvalidDates(t *testing.T) {
	f := &Flat{
		ID:        1,
		Price:     123,
		Status:    "free",
		Updated:   "2024-01-02T03:04:05Z",
		BlockName: "TestBlock",
		BlockSlug: "tb",
		BulkName:  "Корпус 1.1",
		Rooms:     1,
		Area:      10,
		Floor:     1,
		PriceHistory: PriceHistory{
			{Date: "2022-12-31T23:59:59Z", Price: 100, Status: "free"},
			{Date: "invalid-date", Price: 101, Status: "free"},
			{Date: "2023-01-01T00:00:00Z", Price: 110, Status: "free"},
			{Date: "2024-01-01T00:00:00Z", Price: 120, Status: "reserve"},
		},
	}

	h := f.GetPriceHistory()
	require.Len(t, h, 2)
	require.Equal(t, "2023-01-01T00:00:00Z", h[0].Date)
	require.Equal(t, "2024-01-01T00:00:00Z", h[1].Date)
}

