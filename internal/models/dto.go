package models

import "time"

type AdvertiserCount struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

type AnalysisResult struct {
	Domain           string            `json:"domain"`
	TotalAdvertisers int               `json:"total_advertisers"`
	Advertisers      []AdvertiserCount `json:"advertisers"`
	Cached           bool              `json:"cached"`
	Timestamp        time.Time         `json:"timestamp"`
}

type BatchRequest struct {
	Domains []string `json:"domains"`
}
type BatchResponse struct {
	Results []AnalysisResult `json:"results"`
}
