package telegrambot

import (
	"github.com/georgri/pik_tg_bot/pkg/backup_data"
	"github.com/georgri/pik_tg_bot/pkg/downloader"
	"github.com/georgri/pik_tg_bot/pkg/util"
	"log"
	"os"
	"strings"
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

	for _, channelInfo := range ChannelIDs[envType] {
		ProcessWithChannelInfo(channelInfo)
	}
}

func ProcessWithChannelInfo(channelInfo ChannelInfo) {
	chatID := channelInfo.ChatID
	blockSlug := channelInfo.BlockSlug
	blockID := GetBlockIDBySlug(blockSlug)

	flats, filtered, updateCallback, err := downloader.GetFlats(chatID, blockID)
	if err != nil {
		log.Printf("error getting response from pik.ru: %v", err)
		return
	}

	err = updateCallback()
	if err != nil {
		log.Printf("update callback failed in %v (chatID %v): %v", blockSlug, chatID, err)
	}

	if len(strings.TrimSpace(flats)) == 0 {
		log.Printf("No new flats in %v (chatID %v), aborting; filtered %v", blockSlug, chatID, filtered)
		return
	}

	log.Printf("Got flats in %v (chatID %v): %v", blockSlug, chatID, flats)

	err = SendMessage(chatID, flats)
	if err != nil {
		log.Printf("error while sending message in %v (chatID %v): %v", blockSlug, chatID, err)
		return
	}
}
