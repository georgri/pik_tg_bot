package main

import (
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/downloader"
)

const (
	PikUrl = "https://flat.pik-service.ru/api/v1/filter/flat-by-block/1240?type=1,2&location=2,3&flatLimit=80&onlyFlats=1"
)

func main() {
	fmt.Printf("Hello world!\n")

	body, err := downloader.GetUrl(PikUrl)

	if err != nil {
		fmt.Printf("error getting pik url: %v", err)
	}

	fmt.Printf("Body of url %v is:\n%v\n", PikUrl, body)
}
