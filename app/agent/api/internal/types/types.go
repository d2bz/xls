package types

type ChatReq struct {
	Query     string `json:"query"`
	SessionID string `json:"session_id,optional"`
}

type ChatResp struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
	Answer     string `json:"answer"`
	SessionID  string `json:"session_id"`
}

type HealthCheckReq struct{}

type HealthCheckResp struct {
	Status string `json:"status"`
}
