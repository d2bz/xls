package types

type LikeMsg struct {
	UserID     uint64
	TargetID   uint64
	TargetType int32
	IsLike     int32 // 0: 未赞 1: 已赞
}
