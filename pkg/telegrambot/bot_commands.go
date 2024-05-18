package telegrambot

import (
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/util"
	"log"
	"strings"
)

func sendHello(chatID int64, username string) {
	msg := fmt.Sprintf("Hello, %v!", username)
	err := SendMessage(chatID, msg)
	if err != nil {
		log.Printf("failed to send message %v to chatID %v: %v", msg, chatID, err)
	}
}

func sendList(chatID int64) {
	var complexes []string
	for _, comp := range util.SortedKeys(BlockSlugs) {
		complexes = append(complexes, BlockSlugs[comp].String())
	}
	msg := fmt.Sprintf("List of known complexes:\n") + strings.Join(complexes, "\n")
	err := SendMessage(chatID, msg)
	if err != nil {
		log.Printf("failed to send list of all blocks to chatID %v: %v", chatID, err)
	}
}
