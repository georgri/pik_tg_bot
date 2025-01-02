package telegrambot

import (
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/backup_data"
	"github.com/georgri/pik_tg_bot/pkg/downloader"
	"github.com/georgri/pik_tg_bot/pkg/util"
	"log"
	"os"
	"time"
)

const (
	invokeEvery = 5 * time.Minute

	logfile = "logs/bot.log"
)

func RunForever() {
	// set up simple logging
	f, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Printf("error while closing log file: %v", err)
		}
	}(f)

	log.SetOutput(f)

	go backup_data.BackupDataForever()

	go UpdateBlocksForever()

	go GetUpdatesForever()

	for {
		RunOnce()
		time.Sleep(invokeEvery)
	}
}

func RunOnce() {
	envType := util.GetEnvType()

	// 1. Get map of block slug => subscribed channels
	// 2. Update block slug
	// 3. Send info to all subscribed channels

	slugs := make(map[string][]int64, 10)

	for _, channelInfo := range ChannelIDs[envType] {
		slugs[channelInfo.BlockSlug] = append(slugs[channelInfo.BlockSlug], channelInfo.ChatID)
	}
	for slug, chatIDs := range slugs {
		ProcessWithSlugAndChatIDs(slug, chatIDs)
	}
}

func ProcessWithSlugAndChatIDs(blockSlug string, chatIDs []int64) {
	msgs, err := DownloadAndUpdateFile(blockSlug, chatIDs[0])
	if err != nil {
		log.Printf("error while updating flats: %v", err)
		return
	}

	for _, chatID := range chatIDs {
		for _, msg := range msgs {
			err = SendMessage(chatID, msg)
			if err != nil {
				log.Printf("error while sending message in %v (chatID %v): %v", blockSlug, chatID, err)
				return
			}
		}
	}
}

func DownloadAndUpdateFile(blockSlug string, chatID int64) ([]string, error) {
	blockID := GetBlockIDBySlug(blockSlug)

	envtype := util.GetEnvType().String()

	// TODO: get rid of chatIDs[0] after safe migration
	flatMsgs, filtered, updateCallback, err := downloader.GetFlats(chatID, blockID)
	if err != nil {
		return nil, fmt.Errorf("error getting response from pik.ru: %v", err)
	}

	err = updateCallback()
	if err != nil {
		return nil, fmt.Errorf("update callback failed in %v (envtype %v): %v", blockSlug, envtype, err)
	}

	if len(flatMsgs) == 0 {
		return nil, fmt.Errorf("no new flats in %v (envtype %v), aborting; filtered %v", blockSlug, envtype, filtered)
	}

	log.Printf("Got flats in %v (envtype %v): %v", blockSlug, envtype, flatMsgs)

	return flatMsgs, nil
}
