package router

// RouterInput 是 Router Graph 的输入类型。
// 包含用户请求的所有精确参数，在 Graph 内部通过边传递给各节点。
type RouterInput struct {
	Query    string
	UserID   uint64
	VideoID  uint64
	Keyword  string
	Page     int
	PageSize int
}
