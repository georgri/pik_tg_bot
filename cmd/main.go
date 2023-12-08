package main

import (
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/downloader"
	"github.com/georgri/pik_tg_bot/pkg/telegrambot"
	"strings"
)

func main() {
	fmt.Printf("Hello world!\n")

	flats, filtered, updateCallback, err := downloader.GetFlats()
	if err != nil {
		fmt.Printf("error getting pik url: %v", err)
	}

	if len(strings.TrimSpace(flats)) == 0 {
		fmt.Printf("No new flats, aborting; filtered %v", filtered)
		return
	}

	fmt.Printf("Got flats: %v", flats)

	err = telegrambot.SendTestMessage(flats)
	if err != nil {
		fmt.Printf("error while sending message: %v", err)
		return
	}

	err = updateCallback()
	if err != nil {
		fmt.Printf("update callback failed: %v", err)
	}
}
