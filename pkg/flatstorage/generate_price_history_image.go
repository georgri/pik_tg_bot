package flatstorage

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"time"

	"github.com/wcharczuk/go-chart/v2"
)

func generatePriceChart(minHistory, maxHistory PriceHistory) ([]byte, error) {
	minSeries := chart.TimeSeries{
		Name:    "Minimum Prices",
		XValues: make([]time.Time, 0, len(minHistory)),
		YValues: make([]float64, 0, len(minHistory)),
		Style: chart.Style{
			StrokeColor: chart.ColorBlue,
			FillColor:   chart.ColorBlue.WithAlpha(64),
		},
	}

	maxSeries := chart.TimeSeries{
		Name:    "Maximum Prices",
		XValues: make([]time.Time, 0, len(maxHistory)),
		YValues: make([]float64, 0, len(maxHistory)),
		Style: chart.Style{
			StrokeColor: chart.ColorRed,
			FillColor:   chart.ColorRed.WithAlpha(64),
		},
	}

	for _, entry := range minHistory {
		t, err := time.Parse(time.RFC3339, entry.Date)
		if err != nil {
			continue
		}
		minSeries.XValues = append(minSeries.XValues, t)
		minSeries.YValues = append(minSeries.YValues, float64(entry.Price))
	}

	for _, entry := range maxHistory {
		t, err := time.Parse(time.RFC3339, entry.Date)
		if err != nil {
			continue
		}
		maxSeries.XValues = append(maxSeries.XValues, t)
		maxSeries.YValues = append(maxSeries.YValues, float64(entry.Price))
	}

	graph := chart.Chart{
		Title: "Minimum and Maximum Prices Over Time",
		XAxis: chart.XAxis{
			Name:           "Date",
			ValueFormatter: chart.TimeDateValueFormatter,
		},
		YAxis: chart.YAxis{
			Name: "Price (₽)",
		},
		Series: []chart.Series{minSeries, maxSeries},
	}

	var pngBuf bytes.Buffer
	if err := graph.Render(chart.PNG, &pngBuf); err != nil {
		return nil, err
	}

	// Decode PNG
	img, err := png.Decode(&pngBuf)
	if err != nil {
		return nil, err
	}

	// Convert to non-alpha RGB (JPEG does not support transparency)
	rgbImg := image.NewRGBA(img.Bounds())
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			rgbImg.Set(x, y, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: 255,
			})
		}
	}

	var jpgBuf bytes.Buffer
	if err := jpeg.Encode(&jpgBuf, rgbImg, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}

	return jpgBuf.Bytes(), nil
}
