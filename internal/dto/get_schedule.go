package dto

type GetScheduleInput struct {
	From string
	To   string
}

type Day struct {
	Date   string   `json:"date"`
	Booked []string `json:"booked"`
}

type GetScheduleOutput struct {
	Days []Day `json:"days"`
}
