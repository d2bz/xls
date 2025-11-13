package types

type (
	CanalLikeSyncMsg struct {
		Data []struct {
			ID         string `json:"id"`
			UserID     string `json:"user_id"`
			TargetID   string `json:"target_id"`
			TargetType string `json:"target_type"`
		}
		Type string `json:"type"`
	}
	CanalVideoMsg struct {
		Data []struct {
			ID         string `json:"id"`
			Uid        string `json:"uid"`
			Title      string `json:"title"`
			Url        string `json:"url"`
			LikeNum    string `json:"like_num"`
			CommentNum string `json:"comment_num"`
			CreatedAt  string `json:"created_at"`
			UpdatedAt  string `json:"updated_at"`
			DeletedAt  string `json:"deleted_at"`
		}
	}
)
