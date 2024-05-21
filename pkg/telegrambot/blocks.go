package telegrambot

import (
	"fmt"
	"strconv"
	"strings"
)

type BlockInfo struct {
	ID   int64
	Name string
	Slug string
}

type BlockInfoMap map[string]BlockInfo

var BlockSlugs BlockInfoMap

func init() {
	BlockSlugs = make(BlockInfoMap, len(BlockSlugSlice))
	for _, blockSlice := range BlockSlugSlice {
		id, err := strconv.Atoi(blockSlice[0])
		if err != nil {
			panic(err)
		}
		slug := strings.Trim(blockSlice[2], "/")
		BlockSlugs[slug] = BlockInfo{
			ID:   int64(id),
			Slug: slug,
			Name: blockSlice[1],
		}
	}
}

func GetBlockIDBySlug(slug string) int64 {
	blockInfo, _ := BlockSlugs[slug]
	return blockInfo.ID
}

func GetBlockURLBySlug(slug string) string {
	return fmt.Sprintf("https://www.pik.ru/%v", slug)
}

func (b BlockInfo) String() string {
	return fmt.Sprintf("%v: <a href=\"%v\">%v</a>", b.Name, GetBlockURLBySlug(b.Slug), b.Slug)
}
