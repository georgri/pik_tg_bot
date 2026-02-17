package flatstorage

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterWithFlatStorageHelper_PriceDropThreshold_IncludesBoundary(t *testing.T) {
	oldMsg := &MessageData{
		Flats: []Flat{
			{
				ID:        1,
				Price:     100,
				BlockName: "TestBlock",
				BlockSlug: "tb",
				BulkName:  "Корпус 1.1",
				Rooms:     1,
				Area:      10,
				Floor:     1,
			},
		},
	}

	newMsg := &MessageData{
		Flats: []Flat{
			{
				ID:        1,
				Price:     90, // -10%
				BlockName: "TestBlock",
				BlockSlug: "tb",
				BulkName:  "Корпус 1.1",
				Rooms:     1,
				Area:      10,
				Floor:     1,
			},
		},
	}

	res := FilterWithFlatStorageHelper(oldMsg, newMsg)
	require.Len(t, res, 1)
	require.Contains(t, res[0], "flats dropped prices in")
	require.True(t, strings.Contains(res[0], "price-10.0%"), res[0])
}

func TestFilterWithFlatStorageHelper_PriceDropThreshold_ExcludesSmallerDrop(t *testing.T) {
	oldMsg := &MessageData{
		Flats: []Flat{
			{
				ID:        1,
				Price:     100,
				BlockName: "TestBlock",
				BlockSlug: "tb",
				BulkName:  "Корпус 1.1",
				Rooms:     1,
				Area:      10,
				Floor:     1,
			},
		},
	}

	newMsg := &MessageData{
		Flats: []Flat{
			{
				ID:        1,
				Price:     91, // -9%
				BlockName: "TestBlock",
				BlockSlug: "tb",
				BulkName:  "Корпус 1.1",
				Rooms:     1,
				Area:      10,
				Floor:     1,
			},
		},
	}

	res := FilterWithFlatStorageHelper(oldMsg, newMsg)
	require.Len(t, res, 0)
}

