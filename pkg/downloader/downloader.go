package downloader

import (
	"fmt"
	"io"
	"net/http"
)

func GetUrl(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func GetFlats(url string) (string, error) {
	body, err := GetUrl(url)
	if err != nil {
		return "", fmt.Errorf("error while getting url %v: %v", url, err)
	}

	msgData, err := UnmarshallFlats(body)
	if err != nil {
		return "", err
	}

	if len(msgData.flats) == 0 {
		return "", fmt.Errorf("got 0 flats from url")
	}

	// convert flats to human readable message
	msg := msgData.String()

	return msg, nil
}
