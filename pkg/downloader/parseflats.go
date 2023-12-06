package downloader

import "encoding/json"

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
	Price  int64   `json:"price"`
	Rooms  int8    `json:"rooms"`
	Status string  `json:"status"`
	// TODO: find Url plan with areas, maybe with address https://www.pik.ru/flat/819556
	// https://flat.pik-service.ru/api/v1/flat/819556
	PlanURL   string `json:"planUrl"`   // https:\/\/0.db-estate.cdn.pik-service.ru\/layout\/2022\/06\/13\/1_sem2_2el36_4_2x12_6-1_t_a_90_PgbXHE4ZDppCmmc2.svg
	BulkName  string `json:"bulkName"`  // Корпус 1.1
	MaxFloor  int8   `json:"maxFloor"`  // 33
	BlockName string `json:"blockName"` // Второй Нагатинский
}

type Metro struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type Body struct {
	Data Data `json:"data"`
}

type Data struct {
	Items []Flat `json:"items"`
}

func UnmarshallFlats(body []byte) ([]Flat, error) {
	res := &Body{}
	err := json.Unmarshal(body, res)
	if err != nil {
		return nil, err
	}

	return res.Data.Items, nil
}
