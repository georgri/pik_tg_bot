package main

import (
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/downloader"
	"github.com/georgri/pik_tg_bot/pkg/telegrambot"
	"strings"
)

const (
	PikUrl = "https://flat.pik-service.ru/api/v1/filter/flat-by-block/1240?type=1,2&location=2,3&flatLimit=80&onlyFlats=1"
)

func main() {
	fmt.Printf("Hello world!\n")

	flats, err := downloader.GetFlats(PikUrl)
	if err != nil {
		fmt.Printf("error getting pik url: %v", err)
	}

	if len(strings.TrimSpace(flats)) == 0 {
		fmt.Printf("No new flats, aborting")
		return
	}

	fmt.Printf("Got flats: %v", flats)

	err = telegrambot.SendTestMessage(flats)
	if err != nil {
		fmt.Printf("error while sending message: %v", err)
	}
}
