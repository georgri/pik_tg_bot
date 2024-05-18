package telegrambot

import (
	"flag"
	"strconv"
	"strings"
)

type EnvType int8

const (
	EnvTypeDev EnvType = iota
	EnvTypeTesting
	EnvTypeProd
)

var EnvTypeFromString = map[string]EnvType{
	"dev":  EnvTypeDev,
	"test": EnvTypeTesting,
	"prod": EnvTypeProd,
}

var RootEnvType string

func init() {
	flag.StringVar(&RootEnvType, "envtype", "dev", "dev|test|prod")
	flag.Parse()
}

func GetEnvType() EnvType {
	envType, _ := EnvTypeFromString[RootEnvType]
	return envType
}

type ChannelInfo struct {
	ChatID    int64
	BlockSlug string // real estate project, e.g 2ngt, utnv
}

var ChannelIDs = map[EnvType][]ChannelInfo{
	EnvTypeDev: {
		{
			ChatID:    TestChatID,
			BlockSlug: "2ngt",
		},
		{
			ChatID:    TestChatID,
			BlockSlug: "ytnv",
		},
	},
	EnvTypeTesting: {
		{
			ChatID:    TestChatID,
			BlockSlug: "2ngt",
		},
		{
			ChatID:    TestChatID,
			BlockSlug: "ytnv",
		},
	},
	EnvTypeProd: {
		{
			ChatID:    -1001451631453,
			BlockSlug: "2ngt",
		},
		{
			ChatID:    -1001439896663,
			BlockSlug: "sp",
		},
		{
			ChatID:    -1002066659264,
			BlockSlug: "ytnv",
		},
		{
			ChatID:    -1002087536270,
			BlockSlug: "kolskaya8",
		},
		{
			ChatID:    -1002123708132,
			BlockSlug: "hp",
		},

	},
}

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

// TODO: download from website
var BlockSlugSlice = [][]string{
	{"1401", "Первый Дубровский", "/1dubr"},
	{"1240", "Второй Нагатинский", "/2ngt"},
	{"1555", "Алтуфьевское 53", "/alt53"},
	{"481", "Амурский парк", "/amur"},
	{"137", "Академика Павлова", "/apavlova"},
	{"1372", "Большая Академическая 85", "/ba85"},
	{"378", "Барклая 6", "/barclay6"},
	{"294", "Белая Дача парк", "/bd"},
	{"519", "Большая Очаковская 2", "/bo2"},
	{"47", "Бутово парк 2", "/bp2"},
	{"1129", "Бусиновский парк", "/bpark"},
	{"241", "Дмитровский парк", "/dp"},
	{"130", "Green park", "/gp"},
	{"320", "Holland park", "/hp"},
	{"90", "Измайловский лес", "/i-les"},
	{"156", "Ильинские луга", "/il-luga"},
	{"404", "Красноказарменная 15", "/kk15"},
	{"530", "Кольская 8", "/kolskaya8"},
	{"518", "Кронштадтский 9", "/kron9"},
	{"1134", "Кронштадтский 14", "/kron14"},
	{"1196", "Кутузовский квартал", "/kutuzovskiy"},
	{"296", "Кузьминский лес", "/kuzminskyles"},
	{"1272", "Ярославский квартал", "/kvartal-yaroslavskii"},
	{"1421", "Кавказский бульвар 51", "/kvb51"},
	{"132", "Мещерский лес", "/les"},
	{"1460", "Лосиноостровский парк", "/lospark"},
	{"174", "Люберецкий", "/luberecky"},
	{"161", "Люблинский парк", "/lublinpark"},
	{"55", "Бунинские луга", "/luga"},
	{"21", "Восточное Бутово", "/mkr-vostochnoe-butovo"},
	{"1411", "Митинский лес", "/mles"},
	{"253", "Михайловский парк", "/mp"},
	{"1108", "Мичуринский парк", "/mpark"},
	{"1688", "Матвеевский парк", "/mtvpark"},
	{"219", "Мякинино парк", "/myakinino"},
	{"1403", "Новохохловская 15", "/nh15"},
	{"1556", "Никольские луга", "/nluga"},
	{"1692", "Новое Очаково", "/ochakovo"},
	{"149", "Одинцово-1", "/odin"},
	{"1424", "Открытый парк", "/opark"},
	{"544", "Перовское 2", "/perovo2"},
	{"1124", "Полар", "/plr"},
	{"162", "Полярная 25", "/polar"},
	{"301", "Западный порт", "/port"},
	{"172", "Римского-Корсакова 11", "/rk11"},
	{"477", "Руставели 14", "/rustaveli"},
	{"464", "Сигнальный 16", "/s16"},
	{"269", "Шереметьевский", "/sher"},
	{"1200", "Середневский лес", "/sles"},
	{"118", "Саларьево парк", "/sp"},
	{"159", "Столичные поляны", "/spolyany"},
	{"1377", "Сокольнический вал 1", "/sv1"},
	{"1167", "Томилинский бульвар", "/tbulvar"},
	{"419", "Волоколамское 24", "/v24"},
	{"1541", "Vangarden", "/vangarden"},
	{"411", "Волжский парк", "/volp"},
	{"1220", "Яуза парк", "/yauza"},
	{"1369", "Ютаново", "/ytnv"},
	{"1934", "Юнино", "/yunino"},
	{"1519", "Зелёный парк", "/zelpark"},
	{"65", "Ярославский", "/zhiloi-raion-yaroslavskii"},
	{"164", "Жулебино парк", "/zhulebino"},
}
