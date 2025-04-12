package flatstorage

import (
	"encoding/json"
	"fmt"
	"github.com/georgri/pik_tg_bot/pkg/util"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	FlatValidInterval       = 60 * time.Minute
	PriceComparisonInterval = 7 * 24 * time.Hour

	upArrow   = "⬆"
	downArrow = "🔻"

	SimilarAreaThresholdPercent = 2

	PercentageChangeEpsilon = 0.05

	windowPeriod      = 2 * 7 * 24 * time.Hour
	smallWindowPeriod = 24 * time.Hour

	collapsePricesPeriod = 1 * time.Hour
)

// url example: https://flat.pik-service.ru/api/v1/filter/flat-by-block/1240?type=1,2&location=2,3&flatLimit=80&onlyFlats=1
// source example:
// {"id":830713,"area":65.2,"floor":17,"metro":{"id":148,
// "name":"\u041d\u0430\u0433\u0430\u0442\u0438\u043d\u0441\u043a\u0430\u044f","color":"#ACADAF"},
// "price":21796360,"rooms":2,"status":"free","typeId":1,
// "planUrl":"https:\/\/0.db-estate.cdn.pik-service.ru\/layout\/2022\/06\/13\/1_sem2_2el36_4_2x12_6-1_t_a_90_PgbXHE4ZDppCmmc2.svg",
// "bulkName":"\u041a\u043e\u0440\u043f\u0443\u0441 1.1","maxFloor":33,
// "blockName":"\u0412\u0442\u043e\u0440\u043e\u0439 \u041d\u0430\u0433\u0430\u0442\u0438\u043d\u0441\u043a\u0438\u0439",
// "blockSlug":"2ngt","finishType":1,"meterPrice":334300,"settlementDate":"2025-06-15","currentBenefitId":114464}
type Flat struct {
	ID     int64   `json:"id"`
	Area   float64 `json:"area"`
	Floor  int64   `json:"floor"`
	Metro  Metro   `json:"metro"`
	Price  int64   `json:"price"` // in rub
	Rooms  int8    `json:"rooms"`
	Status string  `json:"status"`
	// TODO: find Url plan with areas, maybe with address https://www.pik.ru/flat/819556
	// https://flat.pik-service.ru/api/v1/flat/819556
	PlanURL   string `json:"planUrl"`   // https:\/\/0.db-estate.cdn.pik-service.ru\/layout\/2022\/06\/13\/1_sem2_2el36_4_2x12_6-1_t_a_90_PgbXHE4ZDppCmmc2.svg
	BulkName  string `json:"bulkName"`  // Корпус 1.1
	MaxFloor  int8   `json:"maxFloor"`  // 33
	BlockName string `json:"blockName"` // Второй Нагатинский
	BlockSlug string `json:"blockSlug"`
	Created   string `json:"created,omitempty"` // when the flat first appeared
	Updated   string `json:"updated,omitempty"` // when the flat was last seen (to filter out the old ones)

	FinishType     int8   `json:"finishType"`
	SettlementDate string `json:"settlementDate"`

	AveragePrice int64        `json:"averagePrice"`
	OldPrice     int64        `json:"oldPrice"`
	PriceHistory PriceHistory `json:"priceHistory,omitempty"`
}

type PriceHistory []PriceEntry

type PriceEntry struct {
	Date   string `json:"date,omitempty"` // time.RFC3339
	Price  int64  `json:"price,omitempty"`
	Status string `json:"status,omitempty"`
}

type MessageData struct {
	Flats []Flat `json:"flats"`

	LastPage int
}

type FlatStats struct {
	SimilarFlats []Flat
}

type PriceDropMessageData struct {
	Flats []Flat `json:"flats"`

	PriceDropPercentThreshold int8
}

type Metro struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type Body struct {
	Data Data `json:"data"`
}

type Stats struct {
	LastPage int `json:"lastPage"`
}

type Data struct {
	Items []Flat `json:"items"`
	Stats Stats  `json:"stats"`
}

func UnmarshallFlats(body []byte) (*MessageData, error) {
	unmarshalled := &Body{}
	err := json.Unmarshal(body, unmarshalled)
	if err != nil {
		return nil, err
	}

	res := &MessageData{
		Flats: unmarshalled.Data.Items,

		LastPage: unmarshalled.Data.Stats.LastPage,
	}

	return res, nil
}

func (md *MessageData) Copy() *MessageData {
	if md == nil {
		return nil
	}
	newMsg := &MessageData{LastPage: md.LastPage, Flats: make([]Flat, 0, len(md.Flats))}
	for _, flat := range md.Flats {
		newMsg.Flats = append(newMsg.Flats, flat)
	}
	return newMsg
}

func (f *Flat) GetPriceDropPercentage() float64 {
	if f == nil || f.OldPrice == 0 {
		return 0
	}

	return ((float64(f.Price) / float64(f.OldPrice)) - 1) * 100
}

func (f *Flat) GetPriceBelowAveragePercentage() float64 {
	if f == nil || f.Area == 0 || f.AveragePrice == 0 {
		return 0
	}

	return ((float64(f.Price)/f.Area)/float64(f.AveragePrice) - 1) * 100
}

func (f *Flat) IsSimilar(another Flat) bool {
	if f == nil {
		return false
	}

	if f.Rooms != another.Rooms || f.FinishType != another.FinishType {
		return false
	}

	// area is similar (+-2%)
	return f.Area <= another.Area/100.0*(100.0+SimilarAreaThresholdPercent) && f.Area >= another.Area/100.0*(100.0-SimilarAreaThresholdPercent)
}

func (f *Flat) GetPriceHistory() PriceHistory {
	if f == nil {
		return nil
	}

	if len(f.PriceHistory) == 0 {
		f.PriceHistory = PriceHistory{PriceEntry{
			Date:   f.Updated,
			Price:  f.Price,
			Status: f.Status,
		}}
	}

	size := len(f.PriceHistory)
	if f.PriceHistory[size-1].Status == "" {
		f.PriceHistory[size-1].Status = f.Status
	}

	f.PriceHistory = prunePriceHistory(f.PriceHistory)

	return f.PriceHistory
}

func prunePriceHistory(history PriceHistory) PriceHistory {
	if len(history) == 0 {
		return history
	}

	sort.Slice(history, func(i, j int) bool {
		return history[i].Date < history[j].Date
	})

	result := history[:0]
	i := 0
	for i < len(history) {
		curr := history[i]
		result = append(result, curr)

		// Look for the pattern A -> ... -> A within 1 hour
		var lastAIndex = -1
		startTime, err := time.Parse(time.RFC3339, curr.Date)
		if err != nil {
			i++
			continue
		}

		for j := i + 1; j < len(history); j++ {
			nextTime, err := time.Parse(time.RFC3339, history[j].Date)
			if err != nil {
				break
			}
			if nextTime.Sub(startTime) > collapsePricesPeriod {
				break
			}
			if history[j].Price == curr.Price {
				lastAIndex = j
			}
		}

		if lastAIndex != -1 {
			// Skip all entries between i+1 and lastAIndex inclusive
			i = lastAIndex + 1
		} else {
			i++
		}
	}

	return result
}

// String print in human readable telegram friendly format
// example input:
// {831859 32.6 19 {Нагатинская #ACADAF} 12756380 1 free
// https://0.db-estate.cdn.pik-service.ru/attachment/0/167b4389-02d9-eb11-84e9-02bf0a4d8e27/6_sem2_1es3_5.7-1_s_z_07ef74f33ec511c288fe633c87ef297c.svg
// Корпус 1.3 33 Второй Нагатинский}
// example output:
// {number of Flats} новых объектов в ЖК "Второй Нагатинский" (м.Нагатинская (color #ACADAF)):
// Корпус 1.3 #831859[url link to flat]: 32.6m, 1r, f19, 12_756_380rub,
func (md *MessageData) String() string {
	return md.StringWithOptions(false, false)
}

func (md *MessageData) StringWithOptions(sortByAvg bool, withInfo bool) string {

	if sortByAvg {
		sort.Slice(md.Flats, func(i, j int) bool {
			return md.Flats[i].GetPriceBelowAveragePercentage() < md.Flats[j].GetPriceBelowAveragePercentage()
		})
	} else {
		// sorting by price
		sort.Slice(md.Flats, func(i, j int) bool {
			return md.Flats[i].Price < md.Flats[j].Price
		})
	}

	res := md.MakeHeader()

	flats := make([]string, 0, len(md.Flats))
	for _, flat := range md.Flats {
		flats = append(flats, flat.StringWithOptions())
	}

	res += "\n" + strings.Join(flats, "\n") // try <br>

	return res
}

func (md *MessageData) GetInfoToSend(stats FlatStats) (string, []byte) {
	if len(md.Flats) == 0 {
		return "", nil
	}

	res := fmt.Sprintf("info about flat #%v in compex %v:", md.Flats[0].ID, md.Flats[0].BlockSlug)

	flats := make([]string, 0, len(md.Flats))
	for _, flat := range md.Flats {
		flats = append(flats, flat.String())
		// TODO: format dates and prices nicely
		flats = append(flats, fmt.Sprintf("Price history:"))
		for _, priceEntry := range flat.GetPriceHistory() {
			flats = append(flats, fmt.Sprintf("%v", priceEntry))
		}
	}

	minSeries, maxSeries := CalcPriceMinMaxRangeSeries(stats.SimilarFlats)

	if len(minSeries) == 0 {
		flats = append(flats, fmt.Sprintf("not enough data to calc min/max series :("))
	}

	// TODO: enable later for premium users
	//if len(minSeries) > 0 {
	//	flats = append(flats, fmt.Sprintf("minimum prices for similar reserved flats:"))
	//	for _, pricePoint := range minSeries {
	//		flats = append(flats, fmt.Sprintf("%v", pricePoint))
	//	}
	//}
	//
	//if len(maxSeries) > 0 {
	//	flats = append(flats, fmt.Sprintf("maximum prices for similar reserved flats:"))
	//	for _, pricePoint := range maxSeries {
	//		flats = append(flats, fmt.Sprintf("%v", pricePoint))
	//	}
	//}
	//
	//if len(stats.SimilarFlats) > 0 {
	//	flats = append(flats, fmt.Sprintf("info about similar flats in complex %v:", md.Flats[0].BlockSlug))
	//	for _, flat := range stats.SimilarFlats {
	//		flats = append(flats, flat.String())
	//		// TODO: format dates and prices nicely
	//		flats = append(flats, fmt.Sprintf("%v", flat.GetPriceHistory()))
	//	}
	//}

	res += "\n" + strings.Join(flats, "\n") // try <br>

	img, _ := generatePriceChart(minSeries, maxSeries)

	return res, img
}

func CalcPriceMinMaxRangeSeries(flats []Flat) (PriceHistory, PriceHistory) {
	// make single slice with all the data
	history := make(PriceHistory, 0, len(flats))
	for i := range flats {
		for _, pricePoint := range flats[i].GetPriceHistory() {
			if pricePoint.Status != "reserve" {
				continue
			}
			history = append(history, pricePoint)
		}
	}

	// sort by date
	sort.Slice(history, func(i, j int) bool {
		return history[i].Date < history[j].Date
	})

	var minSeries, maxSeries PriceHistory
	var minDeque, maxDeque PriceHistory
	leftIndex := 0

	// make two weeks window
	for i, pricePoint := range history {
		pointDate, err := time.Parse(time.RFC3339, pricePoint.Date)
		if err != nil {
			log.Printf("failed to parse date %v: %v", pricePoint.Date, err)
			continue
		}
		leftTimeStr := pointDate.Add(-windowPeriod).Format(time.RFC3339)
		smallWindowTimeStr := pointDate.Add(-smallWindowPeriod).Format(time.RFC3339)

		// update minDeque and maxDeque: pop from both while price is lower/higher
		for len(minDeque) > 0 && minDeque[len(minDeque)-1].Price >= pricePoint.Price {
			minDeque = minDeque[:len(minDeque)-1]
		}
		for len(maxDeque) > 0 && maxDeque[len(maxDeque)-1].Price <= pricePoint.Price {
			maxDeque = maxDeque[:len(maxDeque)-1]
		}

		// shift left window pointer, get rid of first elements in deques if needed
		for leftIndex < i && history[leftIndex].Date < leftTimeStr {
			for len(minDeque) > 0 && minDeque[0] == history[leftIndex] {
				minDeque = minDeque[1:]
			}
			for len(maxDeque) > 0 && maxDeque[0] == history[leftIndex] {
				maxDeque = maxDeque[1:]
			}
			leftIndex++
		}

		// push current elem into deques
		minDeque = append(minDeque, pricePoint)
		maxDeque = append(maxDeque, pricePoint)

		// add first elems to minSeries and maxSeries
		if len(minSeries) == 0 || minSeries[len(minSeries)-1] != minDeque[0] {
			if len(minSeries) > 0 && minSeries[len(minSeries)-1].Date > smallWindowTimeStr {
				minSeries[len(minSeries)-1].Price = minDeque[0].Price
			} else {
				minSeries = append(minSeries, minDeque[0])
			}
		}

		if len(maxSeries) == 0 || maxSeries[len(maxSeries)-1] != maxDeque[0] {
			if len(maxSeries) > 0 && maxSeries[len(maxSeries)-1].Date > smallWindowTimeStr {
				maxSeries[len(maxSeries)-1].Price = maxDeque[0].Price
			} else {
				maxSeries = append(maxSeries, maxDeque[0])
			}
		}
	}

	return minSeries, maxSeries
}

// MakeHeader example:
// // {number of Flats} новых объектов в ЖК "Второй Нагатинский" (м.Нагатинская (color #ACADAF)):
func (md *MessageData) MakeHeader() string {

	if md == nil || len(md.Flats) == 0 {
		return ""
	}

	flat := md.Flats[0]
	numFlats := len(md.Flats)
	blockName := flat.BlockName
	// metro := flat.Metro.Name // to large message
	// metroColor := flat.Metro.Color // telegram doesn't support text color :(

	res := fmt.Sprintf("%v new flats in %v:",
		numFlats, blockName)

	return res
}

func (md *MessageData) GetBlockSlug() string {
	if md == nil || len(md.Flats) == 0 {
		return ""
	}
	return md.Flats[0].BlockSlug
}

type AveragePriceKey struct {
	BlockSlug  string
	Rooms      int8
	FinishType int8
}

type AveragePriceAggregator map[AveragePriceKey][]int

func (md *MessageData) CalcAveragePrices() {
	aggregate := make(AveragePriceAggregator)
	for i, flat := range md.Flats {
		key := AveragePriceKey{
			BlockSlug:  flat.BlockSlug,
			Rooms:      flat.Rooms,
			FinishType: flat.FinishType,
		}
		aggregate[key] = append(aggregate[key], i)
	}

	for key := range aggregate {
		priceSum := int64(0)
		squareMeters := float64(0)
		for _, flatIndex := range aggregate[key] {
			priceSum += md.Flats[flatIndex].Price
			squareMeters += md.Flats[flatIndex].Area
		}
		avgPrice := priceSum / int64(squareMeters)

		for _, flatIndex := range aggregate[key] {
			md.Flats[flatIndex].AveragePrice = avgPrice
		}
	}
	return
}

func (f *Flat) RecentlyUpdated(now time.Time) bool {
	t, err := time.Parse(time.RFC3339, f.Updated)
	if err != nil {
		return false
	}
	return now.Sub(t) < FlatValidInterval
}

// String example:
// Корпус 1.3 #831859[url link to flat]: 32.6m, 1r, f19, 12_756_380rub,
func (f *Flat) String() string {
	return f.StringWithOptions()
}

func (f *Flat) StringWithOptions() string {
	if f == nil {
		return ""
	}

	var corp string
	bulkSplit := strings.Split(f.BulkName, " ")
	if len(bulkSplit) > 1 {
		corp = bulkSplit[1]
	} else {
		corp = f.BulkName
	}
	id := f.ID
	flatURL := fmt.Sprintf("https://www.pik.ru/flat/%v", id)
	area := fmt.Sprintf("%.1f", f.Area)
	rooms := f.Rooms
	floor := f.Floor
	price := util.ThousandSep(f.Price, " ")
	var reserve string
	if f.Status == "reserve" {
		reserve = "🔒"
	}

	settlementQuarter := GetSettlementQuarter(f.SettlementDate)

	finishTypeString := GetFinishTypeString(f.FinishType)

	res := fmt.Sprintf("%v: <a href=\"%v\">%vr, %vm2</a>, %vR, f%v%v, %v, %v", corp, flatURL, rooms, area, price, floor, reserve, settlementQuarter, finishTypeString)

	var priceInfo []string
	avgPrice := f.formatAvgPrice()
	if avgPrice != "" {
		priceInfo = append(priceInfo, avgPrice)
	}

	priceChange := f.formatPriceChange()
	if priceChange != "" {
		priceInfo = append(priceInfo, priceChange)
	}

	priceInfoStr := strings.Join(priceInfo, ", ")
	if priceInfoStr == "" {
		priceInfoStr = "info"
	}

	infoCommand := fmt.Sprintf("info_%v_%v", f.BlockSlug, f.ID)
	res += ", " + fmt.Sprintf(`<a href="t.me/%v?start=%v">%v</a>`, util.GetBotUsername(), infoCommand, priceInfoStr)

	return res
}

func (f *Flat) PercentageDropString() string {
	if f == nil {
		return ""
	}

	res := f.String()
	res += fmt.Sprintf(", price%.1f%%", f.GetPriceDropPercentage())

	return res
}

func (f *Flat) formatAvgPrice() string {
	if f.AveragePrice == 0 {
		return ""
	}

	percentage := f.GetPriceBelowAveragePercentage()
	if percentage >= 0 {
		return fmt.Sprintf("avg+%.1f%%", percentage)
	}
	return fmt.Sprintf("avg%.1f%%", percentage)
}

func (f *Flat) formatPriceChange() string {
	priceHistory := f.GetPriceHistory()
	if len(priceHistory) == 0 {
		return ""
	}

	weekAgo := time.Now().Add(-PriceComparisonInterval).Format(time.RFC3339)
	var oldPrice int64
	for _, pricePoint := range priceHistory {
		if pricePoint.Date < weekAgo {
			oldPrice = pricePoint.Price
		}
	}

	if oldPrice == 0 {
		return ""
	}

	percentage := (float64(f.Price)/float64(oldPrice) - 1) * 100
	if percentage > PercentageChangeEpsilon {
		return fmt.Sprintf(upArrow+"%.1f%%", percentage)
	} else if percentage < -PercentageChangeEpsilon {
		return fmt.Sprintf(downArrow+"%.1f%%", -percentage)
	}
	return ""
}

// ex 2025-06-15 => 25Q3
func GetSettlementQuarter(settlementDate string) string {
	if len(settlementDate) < 10 {
		return "сдан"
	}
	year := settlementDate[:4]
	month, err := strconv.Atoi(settlementDate[5:7])
	if err != nil {
		return year
	}
	quarter := ((month - 1) / 3) + 1
	year = year[2:]
	return fmt.Sprintf("%vQ%v", year, quarter)
}

func GetFinishTypeString(finishType int8) string {
	if finishType == 1 {
		return "отделка"
	} else if finishType == 2 {
		return "whitebox"
	}
	return "без отделки"
}

// String print in human readable telegram friendly format
// example input:
// {831859 32.6 19 {Нагатинская #ACADAF} 12756380 1 free
// https://0.db-estate.cdn.pik-service.ru/attachment/0/167b4389-02d9-eb11-84e9-02bf0a4d8e27/6_sem2_1es3_5.7-1_s_z_07ef74f33ec511c288fe633c87ef297c.svg
// Корпус 1.3 33 Второй Нагатинский}
// example output:
// {number of Flats} квартир подешевели более, чем на {price_drop_threshold}% в ЖК "Второй Нагатинский":
// Корпус 1.3 #831859[url link to flat]: 32.6m, 1r, f19, 12_756_380rub, {(price_new/price_old - 1)*100)%
func (md *PriceDropMessageData) String() string {
	return md.StringWithPrompt("flats dropped prices in")
}

func (md *PriceDropMessageData) StringWithPrompt(prompt string) string {
	if md == nil || len(md.Flats) == 0 {
		return ""
	}

	// sorting by percentage drop increasing (-20%, -19%, -15%, etc.)
	sort.Slice(md.Flats, func(i, j int) bool {
		return md.Flats[i].GetPriceDropPercentage() < md.Flats[j].GetPriceDropPercentage()
	})

	res := md.MakeHeader(prompt)

	flats := make([]string, 0, len(md.Flats))
	for _, flat := range md.Flats {
		flats = append(flats, flat.PercentageDropString())
	}

	res += "\n" + strings.Join(flats, "\n") // try <br>

	return res
}

// MakeHeader example:
// {number of Flats} квартир подешевели более, чем на {price_drop_threshold}% в ЖК "Второй Нагатинский":
func (md *PriceDropMessageData) MakeHeader(prompt string) string {

	if md == nil || len(md.Flats) == 0 {
		return ""
	}

	flat := md.Flats[0]
	numFlats := len(md.Flats)
	blockName := flat.BlockName

	res := fmt.Sprintf("%v %v %v:",
		numFlats, prompt, blockName)

	return res
}
