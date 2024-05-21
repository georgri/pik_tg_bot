package telegrambot

import (
	"encoding/json"
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/downloader"
	"log"
	"strings"
	"time"
)

const (
	BlocksURL = "https://flat.pik-service.ru/api/v1/filter/block?type=1,2&blockLimit=1000&geoBox=1.0,179.0-1.0,179.0"

	UpdateBlocksEvery = 1 * time.Hour
)

type BlockSiteData struct {
	Success bool `json:"success"`
	Data    struct {
		Items []struct {
			Id   int64  `json:"id"`
			Name string `json:"name"`
			Path string `json:"path"` // = slug
		} `json:"items"`
	} `json:"data"`
}

func DownloadBlocks() (*BlocksFileData, error) {
	url := BlocksURL
	body, err := downloader.GetUrl(url)
	if err != nil {
		return nil, fmt.Errorf("error while getting url %v: %v", url, err)
	}

	blockSiteData := &BlockSiteData{}
	err = json.Unmarshal(body, blockSiteData)
	if err != nil {
		return nil, err
	}

	blockData := &BlocksFileData{}
	for _, block := range blockSiteData.Data.Items {
		blockData.BlockList = append(blockData.BlockList, BlockInfo{
			ID:   block.Id,
			Name: block.Name,
			Slug: strings.TrimLeft(block.Path, "/"),
		})
	}

	return blockData, nil
}

func UpdateBlocksForever() {
	for {
		UpdateBlocksOnce()
		time.Sleep(UpdateBlocksEvery)
	}
}

func UpdateBlocksOnce() {
	blocks, err := DownloadBlocks()
	if err != nil {
		log.Printf("unable to download blocks: %v", err)
		return
	}

	err = MergeBlocksWithHardcode(blocks)
	if err != nil {
		log.Printf("unable to merge downloaded blocks: %v", err)
		return
	}

	err = SyncBlockStorageToFile()
	if err != nil {
		log.Printf("unable to sync blocks to file: %v", err)
		return
	}
}
