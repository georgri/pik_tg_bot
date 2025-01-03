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
	invokeEvery = 1 * time.Minute

	logfile = "logs/bot.log"
)

var (
	errorNoNewFlats = fmt.Errorf("no new flats")
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

	var count int

	for _, channelInfo := range ChannelIDs[envType] {
		slugs[channelInfo.BlockSlug] = append(slugs[channelInfo.BlockSlug], channelInfo.ChatID)
	}
	for slug, chatIDs := range slugs {
		count += 1
		go ProcessWithSlugAndChatIDs(slug, chatIDs)
	}

	// download info about all the unsubscribed blocks
	for slug := range BlockSlugs {
		if _, ok := slugs[slug]; ok {
			continue // already processed
		}
		count += 1
		go ProcessWithSlugAndChatIDs(slug, nil)
	}
}

func ProcessWithSlugAndChatIDs(blockSlug string, chatIDs []int64) {
	msgs, err := DownloadAndUpdateFile(blockSlug)
	if err != nil {
		if err == errorNoNewFlats {
			return
		}
		log.Printf("error while updating flats: %v", err)
		return
	}

	for i := range msgs {
		if len(msgs[i]) > 0 && msgs[i][0] == '!' { // let ! be the magic symbol to send to all known chats
			err = SendToAllKnownChats(msgs[i])
			if err != nil {
				log.Printf("error while sending message to all known chats about %v: %v", blockSlug, err)
				return
			}
		} else {
			for _, chatID := range chatIDs {
				err = SendMessage(chatID, msgs[i])
				if err != nil {
					log.Printf("error while sending message in %v (chatID %v): %v", blockSlug, chatID, err)
					return
				}
			}
		}
	}
}

func DownloadAndUpdateFile(blockSlug string) ([]string, error) {
	blockID := GetBlockIDBySlug(blockSlug)

	envtype := util.GetEnvType().String()

	flatMsgs, updateCallback, err := downloader.GetFlats(blockID)
	if err != nil {
		if err == downloader.ErrorZeroFlats {
			return nil, errorNoNewFlats
		}
		return nil, fmt.Errorf("error getting response from pik.ru: %v", err)
	}

	err = updateCallback()
	if err != nil {
		return nil, fmt.Errorf("update callback failed in %v (envtype %v): %v", blockSlug, envtype, err)
	}

	if len(flatMsgs) == 0 {
		return nil, errorNoNewFlats
	}

	log.Printf("Got flats in %v (envtype %v): %v", blockSlug, envtype, flatMsgs)

	return flatMsgs, nil
}
