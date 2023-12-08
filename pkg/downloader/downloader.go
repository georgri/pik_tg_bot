package downloader

import (
	"encoding/json"
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/util"
	"io"
	"net/http"
	"os"
)

const fileFlatStorage = "2ngt.json"

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

	if len(msgData.Flats) == 0 {
		return "", fmt.Errorf("got 0 Flats from url")
	}

	// filter through local file (MVP)
	msgData, err = FilterAndUpdateWithFlatStorage(msgData)
	if err != nil {
		return "", fmt.Errorf("err while reading/updating local Flats file: %v", err)
	}

	// convert Flats to human-readable message
	msg := msgData.String()

	return msg, nil
}

// FilterAndUpdateWithFlatStorage filter through local file (MVP)
func FilterAndUpdateWithFlatStorage(msg *MessageData) (*MessageData, error) {
	oldMessageData := &MessageData{}

	// read all from file fileFlatStorage
	content, err := os.ReadFile(fileFlatStorage)
	if err != nil {
		// TODO: handle error somehow?
	} else {
		// unmarshal into json
		err = json.Unmarshal(content, &oldMessageData)
		if err != nil {
			return nil, err
		}
	}

	// gen old map
	oldFlatsMap := make(map[int64]Flat)
	for _, flat := range oldMessageData.Flats {
		oldFlatsMap[flat.ID] = flat
	}

	// filter out existing Flats by ID
	msg.Flats = util.FilterSliceInPlace(msg.Flats, func(i int) bool {
		_, ok := oldFlatsMap[msg.Flats[i].ID]
		return !ok
	})

	// append new Flats to file
	for _, flat := range msg.Flats {
		oldMessageData.Flats = append(oldMessageData.Flats, flat)
	}

	newContent, err := json.Marshal(oldMessageData)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(fileFlatStorage, newContent, 0644)
	if err != nil {
		return nil, err
	}

	return msg, nil
}
