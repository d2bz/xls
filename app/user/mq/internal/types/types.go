package types

type CanalUserInfoMsg struct {
	Data []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Email  string `json:"email"`
		Avatar string `json:"avatar"`
	} `json:"data"`
}
