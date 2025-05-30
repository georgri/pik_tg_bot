package downloader

import (
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/flatstorage"
	"io"
	"net/http"
)

const (
	PikUrl    = "https://flat.pik-service.ru/api/v1/filter/flat-by-block"
	UrlParams = "sortBy=price&orderBy=asc&onlyFlats=1&flatLimit=16"

	flatPageFlag = "flatPage"

	// TODO: download this url to monitor new projects
	BlocksUrl = "https://flat.pik-service.ru/api/v1/filter/block?type=1,2&location=2,3&flatLimit=50&blockLimit=1000&geoBox=55.33638001424489,56.14056105282492-36.96336293218961,38.11418080328337"
)

var ErrorZeroFlats = fmt.Errorf("got zero flats")

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

func GetFlats(blockID int64) (messages []string, updateCallback func() error, err error) {
	url := fmt.Sprintf("%v/%v?%v", PikUrl, blockID, UrlParams)

	msgData, err := GetFlatsSinglePage(url)
	if err != nil {
		return nil, nil, err
	}

	if msgData.LastPage > 1 {
		for i := 2; i <= msgData.LastPage; i++ {
			addUrl := fmt.Sprintf("%v&%v=%v", url, flatPageFlag, i)
			addMsgData, err := GetFlatsSinglePage(addUrl)
			if err != nil {
				return nil, nil, err
			}
			msgData.Flats = append(msgData.Flats, addMsgData.Flats...)
		}
	}

	if len(msgData.Flats) == 0 {
		return nil, nil, ErrorZeroFlats
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
