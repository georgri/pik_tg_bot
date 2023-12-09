package flatstorage

import (
	"encoding/json"
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/util"
	"os"
)

const fileFlatStorage = "data/2ngt.json"

func ReadFlatStorage() (*MessageData, error) {
	msgData := &MessageData{}

	// read all from file fileFlatStorage
	content, err := os.ReadFile(fileFlatStorage)
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
func FilterWithFlatStorage(msg *MessageData) (*MessageData, error) {
	oldMessageData, err := ReadFlatStorage()
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
func UpdateFlatStorage(msg *MessageData) (numAdded int, err error) {
	if msg == nil || len(msg.Flats) == 0 {
		return 0, fmt.Errorf("did not update anything")
	}

	oldMessageData, err := ReadFlatStorage()
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
	err = os.WriteFile(fileFlatStorage, newContent, 0644)
	if err != nil {
		return 0, err
	}

	return numAdded, nil
}
