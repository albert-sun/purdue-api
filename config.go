package purdue_api

// Config contains global settings for use between hooks such as concurrency, etc.
type Config struct {
	Concurrent int `json:"concurrent"` // number of concurrent requests
}
