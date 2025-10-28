package types

const (
	VideoLike = 1 + iota
	CommentLike
)

const (
	MillisecondsPerDay = 86400000
	VideoIDsLength     = 30
)

const (
	LikeKey      = "like#video#"
	HotKey       = "hot#video#24h"
	TempHotKey   = "hot#video#24h#temp"
	TempHotKeyDB = "hot#video#24h#temp#db"
)
