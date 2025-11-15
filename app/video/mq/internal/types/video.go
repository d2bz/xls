package types

type (
	CanalLikeSyncMsg struct {
		Data []struct {
			ID         string `json:"id"`
			UserID     string `json:"user_id"`
			TargetID   string `json:"target_id"`
			TargetType string `json:"target_type"`
		} `json:"data"`
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
		} `json:"data"`
		Type string `json:"type"`
	}

	EsVideoMsg struct {
		VideoID      uint64 `json:"video_id"`
		Title        string `json:"title"`
		Url          string `json:"url"`
		AuthorID     uint64 `json:"author_id"`
		AuthorName   string `json:"author_name"`
		AuthorAvatar string `json:"author_avatar"`
		LikeNum      int64  `json:"like_num"`
		CommentNum   int64  `json:"comment_num"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
		DeletedAt    string `json:"deleted_at"`
	}
)
