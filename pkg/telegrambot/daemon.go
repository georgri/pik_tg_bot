package telegrambot

import (
	"github.com/georgri/pik_tg_bot/pkg/downloader"
	"log"
	"os"
	"strings"
	"time"
)

const (
	invokeEvery = 5 * time.Minute

	logfile = "logs/bot.log"
)

var EnvType string

func GetEnvType() string {
	return EnvType
}

func RunForever(env string) {
	EnvType = env

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

	for {
		RunOnce()
		time.Sleep(invokeEvery)
	}
}

func RunOnce() {
	// TODO: cycle through all chats
	chatID := int64(TestChatID)
	flats, filtered, updateCallback, err := downloader.GetFlats(chatID)
	if err != nil {
		log.Printf("error getting response from pik.ru: %v", err)
		return
	}

	if len(strings.TrimSpace(flats)) == 0 {
		log.Printf("No new flats, aborting; filtered %v", filtered)
		return
	}

	log.Printf("Got flats: %v", flats)

	err = SendTestMessage(flats)
	if err != nil {
		log.Printf("error while sending message: %v", err)
		return
	}

	err = updateCallback()
	if err != nil {
		log.Printf("update callback failed: %v", err)
	}
}
