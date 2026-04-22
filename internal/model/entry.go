package model

type Entry struct {
	ID        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Date      string `json:"date"`
	Subject   string `json:"subject"`
	Minutes   int    `json:"minutes"`
}
