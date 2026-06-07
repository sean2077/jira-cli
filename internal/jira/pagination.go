package jira

type Page struct {
	StartAt     int `json:"startAt"`
	MaxResults  int `json:"maxResults"`
	Total       int `json:"total"`
	NextStartAt int `json:"nextStartAt,omitempty"`
}

func NextStartAt(startAt, total, itemCount int) (int, bool) {
	if itemCount <= 0 {
		return 0, false
	}
	next := startAt + itemCount
	if total >= 0 && next >= total {
		return 0, false
	}
	return next, true
}
