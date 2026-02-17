package main

import (
	"flag"
	"github.com/georgri/pik_tg_bot/pkg/telegrambot"
)

func main() {
	flag.Parse()
	telegrambot.RunForever()
}
