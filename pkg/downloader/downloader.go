package downloader

import (
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/flatstorage"
	"io"
	"net/http"
)

const (
	PikUrl = "https://flat.pik-service.ru/api/v1/filter/flat-by-block/1240?sortBy=price&orderBy=asc&onlyFlats=1&flatLimit=16"

	flatPageFlag = "flatPage"
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

func GetFlatsSinglePage(url string) (*flatstorage.MessageData, error) {
	body, err := GetUrl(url)
	if err != nil {
		return nil, fmt.Errorf("error while getting url %v: %v", url, err)
	}

	msgData, err := flatstorage.UnmarshallFlats(body)
	if err != nil {
		return nil, err
	}

	return msgData, nil
}

func GetFlats(chatID int64) (message string, filtered int, updateCallback func() error, err error) {
	url := PikUrl

	msgData, err := GetFlatsSinglePage(url)
	if err != nil {
		return "", 0, nil, err
	}

	if msgData.LastPage > 1 {
		for i := 2; i <= msgData.LastPage; i++ {
			addUrl := fmt.Sprintf("%v&%v=%v", url, flatPageFlag, i)
			addMsgData, err := GetFlatsSinglePage(addUrl)
			if err != nil {
				return "", 0, nil, err
			}
			msgData.Flats = append(msgData.Flats, addMsgData.Flats...)
		}

	}

	if len(msgData.Flats) == 0 {
		return "", 0, nil, fmt.Errorf("got 0 Flats from url")
	}

	// filter through local file (MVP)
	sizeBefore := len(msgData.Flats)
	msgData, err = flatstorage.FilterWithFlatStorage(msgData, chatID)
	if err != nil {
		return "", 0, nil, fmt.Errorf("err while reading/updating local Flats file: %v", err)
	}

	// convert Flats to human-readable message
	msg := msgData.String()

	updateCallback = func() error {
		_, err = flatstorage.UpdateFlatStorage(msgData, chatID)
		return err
	}

	return msg, sizeBefore - len(msgData.Flats), updateCallback, nil
}
