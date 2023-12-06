package downloader

import (
	"fmt"
	"io"
	"net/http"
)

func GetUrl(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	flats, err := UnmarshallFlats(body)
	if err != nil {
		return "", err
	}

	if len(flats) == 0 {
		return "", fmt.Errorf("got 0 flats from url")
	}

	return fmt.Sprintf("List of flats: %v", flats), nil
}
