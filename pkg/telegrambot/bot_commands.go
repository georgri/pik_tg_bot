package telegrambot

import (
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/flatstorage"
	"github.com/georgri/pik_tg_bot/pkg/util"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	DumpCommand        = "dump"
	DumpAvgCommand     = "dumpavg"
	DumpInfoCommand    = "dumpinfo"
	SubscribeCommand   = "sub"
	UnsubscribeCommand = "unsub"
	InfoCommand        = "info"
)

func sendHello(chatID int64, username string) {
	msg := fmt.Sprintf("Hello, %v!", username)
	err := SendMessage(chatID, msg)
	if err != nil {
		log.Printf("failed to send message %v to chatID %v: %v", msg, chatID, err)
	}
}

func sendList(chatID int64, command string) {

	subscribedTo := GetChatSubscriptions(chatID)

	var complexes []string
	for _, comp := range util.SortedKeysByFunc(BlockSlugs, func(a, b string) bool {
		return BlockSlugs[a].Name < BlockSlugs[b].Name
	}) {
		if strings.Contains(command, "dump") {
			complexes = append(complexes, BlockSlugs[comp].StringWithCommand(command))
		} else {
			isSubscribed := subscribedTo[comp]
			complexes = append(complexes, BlockSlugs[comp].StringWithSub(isSubscribed))
		}
	}
	msg := fmt.Sprintf("List of known complexes:\n") + strings.Join(complexes, "\n")
	err := SendMessage(chatID, msg)
	if err != nil {
		log.Printf("failed to send list of all blocks to chatID %v: %v", chatID, err)
	}
}

func GetChatSubscriptions(chatID int64) map[string]bool {
	envtype := util.GetEnvType()
	res := make(map[string]bool, 10)
	for _, channel := range ChannelIDs[envtype] {
		if channel.ChatID == chatID {
			res[channel.BlockSlug] = true
		}
	}
	return res
}

func validateSlug(chatID int64, slug string, command string) (string, error) {
	slug = strings.TrimLeft(strings.TrimSpace(slug), "/")

	_, slugIsValid := BlockSlugs[util.EmbedSlug(slug)]

	if len(slug) == 0 || !slugIsValid {
		sendList(chatID, command)
		return "", fmt.Errorf("slug is empty or invalid: %v", slug)
	}
	return slug, nil
}

func sendDump(chatID int64, slug string, command string) {

	slug, err := validateSlug(chatID, slug, command)
	if err != nil {
		log.Printf("failed to dump to %v: %v", chatID, err)
		return
	}

	var msg string

	// send all known flats for complex with slug "slug"
	fileName := flatstorage.GetStorageFileNameByBlockSlug(slug)
	if !flatstorage.FileExists(fileName) {
		log.Printf("failed to dump flats for slug %v: %v", slug, err)
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

	msg = allFlatsMessageData.StringWithOptions(command == DumpAvgCommand, command == DumpInfoCommand)
	if len(allFlatsMessageData.Flats) == 0 {
		msg = fmt.Sprintf("No known flats for complex %v", slug)
	}

	SendMessageWithPinAsync(chatID, msg, true)
}

func sendInfo(chatID int64, slugAndFlatID string, command string) {

	split := strings.Split(slugAndFlatID, "_")
	flatIDStr := split[len(split)-1]
	slug, _ := strings.CutSuffix(slugAndFlatID, "_"+flatIDStr)

	flatID, err := strconv.ParseInt(flatIDStr, 10, 64)
	if err != nil {
		log.Printf("failed to parse flatID %v from %v: %v", flatIDStr, slugAndFlatID, err)
	}

	slug, err = validateSlug(chatID, slug, command)
	if err != nil {
		log.Printf("failed to send info to %v about %v: %v", chatID, slugAndFlatID, err)
		return
	}

	var msg string

	// send info about flat with given ID
	fileName := flatstorage.GetStorageFileNameByBlockSlug(slug)
	if !flatstorage.FileExists(fileName) {
		log.Printf("failed to get filename for slug %v: %v", slug, err)
	}
	allFlatsMessageData, err := flatstorage.ReadFlatStorage(fileName)
	if err != nil {
		log.Printf("failed to read file with flats %v: %v", fileName, err)
		return
	}

	// save orig slice for calculating stats
	allFlats := allFlatsMessageData.Flats

	// select only matching flats
	allFlatsMessageData.Flats = util.FilterSliceInPlace(allFlatsMessageData.Flats, func(i int) bool {
		return allFlatsMessageData.Flats[i].ID == flatID
	})

	if len(allFlatsMessageData.Flats) == 0 {
		msg = fmt.Sprintf("No flats found with ID %v in complex %v", flatID, slug)
		SendMessageWithPinAsync(chatID, msg, false)
		return
	}

	stats := flatstorage.FlatStats{}
	for _, flat := range allFlats[len(allFlatsMessageData.Flats):] {
		if flat.IsSimilar(allFlatsMessageData.Flats[0]) {
			stats.SimilarFlats = append(stats.SimilarFlats, flat)
		}
	}
	msg, img := allFlatsMessageData.GetInfoToSend(stats)

	SendMessageWithImgAsync(chatID, msg, img, "min and max prices for similar flats", false)
}

func AddNewSubscriber(chatID int64, slug string) error {
	envtype := util.GetEnvType()
	ChannelIDs[envtype] = append(ChannelIDs[envtype], ChannelInfo{
		ChatID:    chatID,
		BlockSlug: slug,
	})

	err := SyncChannelStorageToFile()
	if err != nil {
		n := len(ChannelIDs[envtype])
		ChannelIDs[envtype] = ChannelIDs[envtype][:n-1]
		return err
	}

	return nil
}

func RemoveSubscriber(chatID int64, slug string) error {
	envtype := util.GetEnvType()

	indexToRemove := -1
	for i, subscription := range ChannelIDs[envtype] {
		if subscription.BlockSlug == slug && subscription.ChatID == chatID {
			indexToRemove = i
			break
		}
	}
	if indexToRemove < 0 || indexToRemove >= len(ChannelIDs[envtype]) {
		return fmt.Errorf("chat %v was not subscribed to %v", chatID, slug)
	}

	ChannelIDs[envtype] = util.RemoveSliceElement(ChannelIDs[envtype], indexToRemove)

	err := SyncChannelStorageToFile()
	if err != nil {
		return err
	}

	return nil
}

func CheckSubscribed(chatID int64, slug string) bool {
	envtype := util.GetEnvType()

	for _, subscription := range ChannelIDs[envtype] {
		if subscription.BlockSlug == slug && subscription.ChatID == chatID {
			return true
		}
	}
	return false
}

func subscribeChat(chatID int64, slug string) {
	slug = util.EmbedSlug(slug)

	slug, err := validateSlug(chatID, slug, SubscribeCommand)
	if err != nil {
		log.Printf("failed to subscribe %v to slug %v: %v", chatID, slug, err)
		return
	}

	embeddedSlug := util.EmbedSlug(slug)

	if CheckSubscribed(chatID, slug) {
		// send already subscribed message
		err = SendMessage(chatID, fmt.Sprintf("You are already subscribed to complex %v.\n"+
			"To view all flats: /%v_%v", slug, DumpCommand, embeddedSlug))
		if err != nil {
			log.Printf("failed to send already subscribed message to %v: %v", chatID, err)
		}
		log.Printf("chat %v is already subscribed to %v", chatID, slug)
		return
	}

	err = AddNewSubscriber(chatID, slug)
	if err != nil {
		// send something went wrong while subscribing message
		err = SendMessage(chatID, fmt.Sprintf("Something went wrong while subscribing to %v:\n"+
			"error: %v\n"+
			"You can try again later with /%v_%v", slug, err, SubscribeCommand, embeddedSlug))
		if err != nil {
			log.Printf("failed to send subscription failed message to %v: %v", chatID, err)
		}
		log.Printf("failed to subscribe %v to %v", chatID, slug)
		return
	}

	// send message "You are subscribed"
	err = SendMessage(chatID, fmt.Sprintf("You are now subscribed to new flats from: %v.\n"+
		"To unsubscribe, click here: /%v_%v\n"+
		"To get all known flats click here: /%v_%v", slug, UnsubscribeCommand, embeddedSlug, DumpCommand, embeddedSlug))
	if err != nil {
		log.Printf("failed to send subscribed message to %v: %v", chatID, err)
	}
}

func unsubscribeChat(chatID int64, slug string) {
	slug, err := validateSlug(chatID, slug, UnsubscribeCommand)
	if err != nil {
		log.Printf("failed to unsubscribe %v from slug %v: %v", chatID, slug, err)
		return
	}

	embeddedSlug := util.EmbedSlug(slug)

	if !CheckSubscribed(chatID, slug) {
		// send already subscribed message
		err = SendMessage(chatID, fmt.Sprintf("You are not currently subscribed to complex %v.\n"+
			"To subscribe: /%v_%v\n"+
			"To view all flats: /%v_%v", slug, DumpCommand, embeddedSlug, SubscribeCommand, embeddedSlug))
		if err != nil {
			log.Printf("failed to send already unsubscribed message to %v: %v", chatID, err)
		}
		log.Printf("chat %v is already unsubscribed to %v", chatID, slug)
		return
	}

	err = RemoveSubscriber(chatID, slug)
	if err != nil {
		// send something went wrong while unsubscribing message
		err = SendMessage(chatID, fmt.Sprintf("Something went wrong while unsubscribing from %v:\n"+
			"error: %v\n"+
			"You might need to try again later with /%v_%v", slug, err, UnsubscribeCommand, embeddedSlug))
		if err != nil {
			log.Printf("failed to send unsubscription failed message to %v: %v", chatID, err)
		}
		log.Printf("failed to unsubscribe %v from %v", chatID, slug)
		return
	}

	// send message "You are unsubscribed"
	err = SendMessage(chatID, fmt.Sprintf("You were unsubscribed from: %v.\n"+
		"To subscribe again, click here: /%v_%v\n"+
		"To get all known flats click here: /%v_%v", slug, SubscribeCommand, embeddedSlug, DumpCommand, embeddedSlug))
	if err != nil {
		log.Printf("failed to send unsubscribed message to %v: %v", chatID, err)
	}
}
