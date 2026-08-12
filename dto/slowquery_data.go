package dto

type SlowQueryData struct {
	ID         int64  `json:"id"`
	Query      string `json:"query"`
	DurationMs int64  `json:"duration_ms"`
	CreatedAt  string `json:"created_datetime"`
}

type SlowQueryRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
