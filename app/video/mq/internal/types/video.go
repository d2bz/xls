package types

type CanalMsg struct {
	Data []struct {
		ID         string `json:"id"`
		UserID     string `json:"user_id"`
		TargetID   string `json:"target_id"`
		TargetType string `json:"target_type"`
	}
	Type string `json:"type"`
}
