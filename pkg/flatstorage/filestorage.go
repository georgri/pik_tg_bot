package flatstorage

import (
	"encoding/json"
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/util"
	"os"
)

const (
	storageDir    = "data"
	storageFormat = "json"
)

func ReadFlatStorage(fileName string) (*MessageData, error) {
	msgData := &MessageData{}

	// read all from file fileFlatStorage
	content, err := os.ReadFile(fileName)
	if err != nil {
		// TODO: handle error somehow?
	} else {
		// unmarshal into json
		err = json.Unmarshal(content, &msgData)
		if err != nil {
			return nil, err
		}
	}

	return msgData, nil
}

// FilterWithFlatStorage filter through local file (MVP)
func FilterWithFlatStorage(msg *MessageData, chatID int64) (*MessageData, error) {
	if msg == nil || len(msg.Flats) == 0 {
		return msg, nil
	}

	storageFileName := GetStorageFileName(msg, chatID)
	oldMessageData, err := ReadFlatStorage(storageFileName)
	if err != nil {
		return nil, err
	}

	msg = FilterWithFlatStorageHelper(oldMessageData, msg)

	return msg, nil
}

func FilterWithFlatStorageHelper(oldMsg, newMsg *MessageData) *MessageData {
	// gen old map
	oldFlatsMap := make(map[int64]Flat)
	for _, flat := range oldMsg.Flats {
		oldFlatsMap[flat.ID] = flat
	}

	// filter out existing Flats by ID
	newMsg.Flats = util.FilterSliceInPlace(newMsg.Flats, func(i int) bool {
		_, ok := oldFlatsMap[newMsg.Flats[i].ID]
		return !ok
	})

	newMsg.Flats = util.FilterUnique(newMsg.Flats, func(i int) int64 {
		return newMsg.Flats[i].ID
	})

	return newMsg
}

// UpdateFlatStorage update local file (MVP)
func UpdateFlatStorage(msg *MessageData, chatID int64) (numAdded int, err error) {
	if msg == nil || len(msg.Flats) == 0 {
		return 0, fmt.Errorf("did not update anything")
	}

	storageFileName := GetStorageFileName(msg, chatID)
	oldMessageData, err := ReadFlatStorage(storageFileName)
	if err != nil {
		return 0, err
	}

	msg = FilterWithFlatStorageHelper(oldMessageData, msg)

	numAdded = len(msg.Flats)

	// append new Flats to file
	for _, flat := range msg.Flats {
		oldMessageData.Flats = append(oldMessageData.Flats, flat)
	}

	newContent, err := json.Marshal(oldMessageData)
	if err != nil {
		return 0, err
	}
	err = os.WriteFile(storageFileName, newContent, 0644)
	if err != nil {
		return 0, err
	}

	return numAdded, nil
}

func GetStorageFileName(msg *MessageData, chatID int64) string {
	blockSlug := msg.GetBlockSlug()
	return GetStorageFileNameByBlockSlugAndChatID(blockSlug, chatID)
}

func GetStorageFileNameByBlockSlugAndChatID(blockSlug string, chatID int64) string {
	return fmt.Sprintf("%v/%v_%v.%v", storageDir, blockSlug, chatID, storageFormat)
}
