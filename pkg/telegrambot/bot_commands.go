package telegrambot

import (
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/flatstorage"
	"github.com/georgri/pik_tg_bot/pkg/util"
	"log"
	"strings"
	"time"
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

func sendDump(chatID int64, slug string) {

	if len(slug) == 0 {
		// send help message
		err := SendMessage(chatID, "usage: /dump [code]\nTo get [code] of any complex type /list")
		if err != nil {
			log.Printf("failed to send /dump help message: %v", err)
			return
		}
		return
	}

	// send all known flats for complex with slug "slug"
	slug = strings.TrimLeft(strings.TrimSpace(slug), "/")
	fileName, err := GetStorageFileNameByBlockSlug(slug)
	if err != nil {
		log.Printf("failed to get filename for blockslug %v: %v", slug, err)
		return
	}

	allFlatsMessageData, err := flatstorage.ReadFlatStorage(fileName)
	if err != nil {
		log.Printf("failed to read file with flats %v: %v", fileName, err)
		return
	}

	// output recently updated only
	now := time.Now()
	allFlatsMessageData.Flats = util.FilterSliceInPlace(allFlatsMessageData.Flats, func(i int) bool {
		return allFlatsMessageData.Flats[i].RecentlyUpdated(now)
	})

	msg := allFlatsMessageData.String()
	if len(allFlatsMessageData.Flats) == 0 {
		msg = fmt.Sprintf("No known flats for complex %v", slug)
	}
	err = SendMessage(chatID, msg)
	if err != nil {
		log.Printf("failed to send list of all blocks to chatID %v: %v", chatID, err)
	}
}

func GetStorageFileNameByBlockSlug(blockSlug string) (string, error) {
	// guess chatID
	// TODO: go with empty chatID
	var chatID int64
	for _, channel := range ChannelIDs[GetEnvType()] {
		if channel.BlockSlug == blockSlug {
			chatID = channel.ChatID
			break
		}
	}
	if chatID == 0 {
		return "", fmt.Errorf("yet unknown block slug: %v", blockSlug)
	}
	return flatstorage.GetStorageFileNameByBlockSlugAndChatID(blockSlug, chatID), nil
}
