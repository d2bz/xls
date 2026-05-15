package types

type CanalTagMsg struct {
	Data []struct {
		ID      string `json:"id"`
		TagName string `json:"tag_name"`
	} `json:"data"`
	Type string `json:"type"`
}
