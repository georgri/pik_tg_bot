package flatstorage

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/util"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	FileMaxUpdatePeriod = 30 * time.Minute

	storageDir    = "data"
	storageFormat = "json"

	DefaultPriceDropPercentThreshold        = 15
	DefaultExtremePriceDropPercentThreshold = 20
)

var FileMutex sync.RWMutex

func ReadFlatStorage(fileName string) (*MessageData, error) {
	msgData := &MessageData{}

	FileMutex.RLock()
	defer FileMutex.RUnlock()

	if !FileExistsNonBlocking(fileName) {
		return msgData, nil
	}

	// read all from file fileFlatStorage
	content, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
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
func FilterWithFlatStorage(msg *MessageData) ([]string, error) {
	if msg == nil || len(msg.Flats) == 0 {
		return []string{msg.String()}, nil
	}

	storageFileName := GetStorageFileName(msg)
	oldMessageData, err := ReadFlatStorage(storageFileName)
	if err != nil {
		return nil, err
	}

	return FilterWithFlatStorageHelper(oldMessageData, msg), nil
}

func FilterWithFlatStorageHelper(oldMsg, newMsg *MessageData) []string {
	// gen old map
	oldFlatsMap := make(map[int64]int)
	for oldIndex := range oldMsg.Flats {
		oldFlatsMap[oldMsg.Flats[oldIndex].ID] = oldIndex
	}

	var priceDropList []Flat
	var extremePriceDropList []Flat
	for i := range newMsg.Flats {
		oldIndex, ok := oldFlatsMap[newMsg.Flats[i].ID]
		if !ok {
			// check if price of new flat is below average
			if newMsg.Flats[i].GetPriceBelowAveragePercentage() <= -DefaultExtremePriceDropPercentThreshold {
				extremePriceDropList = append(extremePriceDropList, newMsg.Flats[i])
			}
			continue // skip new flats
		}
		newMsg.Flats[i].OldPrice = oldMsg.Flats[oldIndex].Price
		if newMsg.Flats[i].GetPriceDropPercentage() <= -DefaultExtremePriceDropPercentThreshold {
			extremePriceDropList = append(extremePriceDropList, newMsg.Flats[i])
		} else if newMsg.Flats[i].GetPriceDropPercentage() <= -DefaultPriceDropPercentThreshold {
			priceDropList = append(priceDropList, newMsg.Flats[i])
		}
	}

	var priceDropMsg *PriceDropMessageData
	if len(priceDropList) > 0 {
		priceDropMsg = &PriceDropMessageData{
			Flats:                     priceDropList,
			PriceDropPercentThreshold: DefaultPriceDropPercentThreshold,
		}
	}

	var extremePriceDropMsg *PriceDropMessageData
	if len(extremePriceDropList) > 0 {
		extremePriceDropMsg = &PriceDropMessageData{
			Flats:                     extremePriceDropList,
			PriceDropPercentThreshold: DefaultExtremePriceDropPercentThreshold,
		}
	}

	// filter out existing Flats by ID
	newMsg.Flats = util.FilterSliceInPlace(newMsg.Flats, func(i int) bool {
		_, ok := oldFlatsMap[newMsg.Flats[i].ID]
		return !ok
	})

	newMsg.Flats = util.FilterUnique(newMsg.Flats, func(i int) int64 {
		return newMsg.Flats[i].ID
	})

	var res []string
	msgStr := newMsg.String()
	if len(strings.TrimSpace(msgStr)) > 0 {
		res = append(res, msgStr)
	}

	priceDropStr := priceDropMsg.String()
	if len(strings.TrimSpace(priceDropStr)) > 0 {
		res = append(res, priceDropStr)
	}

	extremePriceDropStr := extremePriceDropMsg.StringWithPrompt(fmt.Sprintf("extreme price drops in"))
	if len(strings.TrimSpace(extremePriceDropStr)) > 0 {
		res = append(res, "!!! "+extremePriceDropStr) // add magic symbol to send to all known chats
	}

	return res
}

type oldFlatInfo struct {
	Created      string
	OldPrice     int64
	PriceHistory []PriceEntry
}

func MergeNewFlatsIntoOld(oldMsg, newMsg *MessageData) *MessageData {
	newMsg.Flats = util.FilterUnique(newMsg.Flats, func(i int) int64 {
		return newMsg.Flats[i].ID
	})

	// gen new map
	newFlatsMap := make(map[int64]struct{})
	for i := range newMsg.Flats {
		newFlatsMap[newMsg.Flats[i].ID] = struct{}{}
	}

	now := time.Now().Format(time.RFC3339)
	past := time.Now().Add(-10 * 365 * 24 * time.Hour).Format(time.RFC3339)

	// map with old flats info
	oldFlatsMap := make(map[int64]oldFlatInfo)
	for i := range oldMsg.Flats {
		if len(oldMsg.Flats[i].Created) == 0 {
			oldMsg.Flats[i].Created = past
		}
		if len(oldMsg.Flats[i].Updated) == 0 {
			oldMsg.Flats[i].Updated = oldMsg.Flats[i].Created
		}
		oldFlatsMap[oldMsg.Flats[i].ID] = oldFlatInfo{
			Created:      oldMsg.Flats[i].Created,
			OldPrice:     oldMsg.Flats[i].Price,
			PriceHistory: oldMsg.Flats[i].PriceHistory,
		}
	}

	// filter out existing old Flats by ID
	oldMsg.Flats = util.FilterSliceInPlace(oldMsg.Flats, func(i int) bool {
		_, ok := newFlatsMap[oldMsg.Flats[i].ID]
		return !ok
	})

	// update both "Created" and "Updated" for downloaded flats
	for i := range newMsg.Flats {
		newMsg.Flats[i].Created = now
		if oldInfo, ok := oldFlatsMap[newMsg.Flats[i].ID]; ok {
			newMsg.Flats[i].Created = oldInfo.Created
			newMsg.Flats[i].OldPrice = oldInfo.OldPrice
			newMsg.Flats[i].PriceHistory = oldInfo.PriceHistory

			if newMsg.Flats[i].Price != newMsg.Flats[i].OldPrice || len(newMsg.Flats[i].PriceHistory) == 0 {
				newMsg.Flats[i].PriceHistory = append(newMsg.Flats[i].PriceHistory, PriceEntry{
					Date:  now,
					Price: newMsg.Flats[i].Price,
				})
			}
		} else {
			// for new flats always add the current price
			newMsg.Flats[i].PriceHistory = append(newMsg.Flats[i].PriceHistory, PriceEntry{
				Date:  now,
				Price: newMsg.Flats[i].Price,
			})
		}
		newMsg.Flats[i].Updated = now
	}

	// dump new into old
	oldMsg.Flats = append(oldMsg.Flats, newMsg.Flats...)

	return oldMsg
}

// UpdateFlatStorage update local file (MVP)
func UpdateFlatStorage(msg *MessageData) (numUpdated int, err error) {
	if msg == nil || len(msg.Flats) == 0 {
		return 0, fmt.Errorf("did not update anything")
	}

	storageFileName := GetStorageFileNameByEnv(msg)
	oldMessageData, err := ReadFlatStorage(storageFileName)
	if err != nil {
		return 0, err
	}

	oldMessageData = MergeNewFlatsIntoOld(oldMessageData, msg)

	numUpdated = len(msg.Flats)

	newStorageFileName := GetStorageFileNameByEnv(msg)
	newContent, err := json.Marshal(oldMessageData)
	if err != nil {
		return 0, err
	}

	FileMutex.Lock()
	defer FileMutex.Unlock()

	err = os.WriteFile(newStorageFileName, newContent, 0644)
	if err != nil {
		return 0, err
	}

	return numUpdated, nil
}

func GetStorageFileNameByEnv(msg *MessageData) string {
	blockSlug := msg.GetBlockSlug()
	return GetStorageFileNameByBlockSlugAndEnv(blockSlug)
}

func GetStorageFileName(msg *MessageData) string {
	blockSlug := msg.GetBlockSlug()
	return GetStorageFileNameByBlockSlug(blockSlug)
}

func GetStorageFileNameByBlockSlugAndEnv(blockSlug string) string {
	return fmt.Sprintf("%v/%v_%v.%v", storageDir, blockSlug, util.GetEnvType().String(), storageFormat)
}

func GetStorageFileNameByBlockSlug(blockSlug string) string {
	// First, try find file without any chatID but with envtype
	targetFileName := GetStorageFileNameByBlockSlugAndEnv(blockSlug)
	if FileExists(targetFileName) {
		return targetFileName
	}
	fileNameWithChatID := fmt.Sprintf("%v/%v_%v.%v", storageDir, blockSlug, 0, storageFormat)
	if FileExists(fileNameWithChatID) {
		return fileNameWithChatID
	}
	return targetFileName
}

func FileExists(filename string) bool {
	return FileExistsNonBlocking(filename)
}

func FileExistsNonBlocking(filename string) bool {
	_, err := os.Stat(filename)
	return !errors.Is(err, os.ErrNotExist)
}

func FileNotUpdated(filename string) bool {
	stat, err := os.Stat(filename)
	if err != nil {
		return true
	}
	now := time.Now()
	fileModified := stat.ModTime()
	return now.Sub(fileModified) > FileMaxUpdatePeriod
}
