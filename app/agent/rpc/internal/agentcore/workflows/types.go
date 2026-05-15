package workflows

import "strings"

type Intent string

const (
	IntentVideoSearch        Intent = "video_search"
	IntentVideoAnalysis     Intent = "video_analysis"
	IntentUserRelation      Intent = "user_relation"
	IntentRecommend         Intent = "recommend"
	IntentRecommendSemantic Intent = "recommend_semantic"
	IntentComplexAnalysis  Intent = "complex_analysis"
	IntentGeneral          Intent = "general"
	IntentFallback         Intent = "fallback"
	// IntentSimple 合并了 video_search、recommend、user_relation 三个简单意图，
	// 统一由 ReAct 工作流处理。supervisor 仍按原意图分类，分支处统一路由到 simple_task。
	IntentSimple Intent = "simple"
)

type SemanticDim struct {
	Name   string   `json:"name,omitempty"`
	Tags   []string `json:"tags,omitempty"`
	Weight int      `json:"weight,omitempty"`
}

type TaskSlot struct {
	Keyword   string          `json:"keyword,omitempty"`
	Sort      string          `json:"sort,omitempty"`
	Limit     int             `json:"limit,omitempty"`
	Page      int             `json:"page,omitempty"`
	UserID    uint64          `json:"user_id,omitempty"`
	VideoID   uint64          `json:"video_id,omitempty"`
	AuthorID  uint64          `json:"author_id,omitempty"`
	TargetUID uint64          `json:"target_uid,omitempty"`
	Dims      []*SemanticDim  `json:"dims,omitempty"`
}

type Complexity string

const (
	ComplexitySimple   Complexity = "simple"
	ComplexityMedium  Complexity = "medium"
	ComplexityComplex Complexity = "complex"
)

type Task struct {
	Query     string        `json:"query,omitempty"`
	Intent    Intent        `json:"intent"`
	Slots     *TaskSlot     `json:"slots"`
	Complexity Complexity    `json:"complexity"`
	Confidence float64      `json:"confidence"`
}

// VideoInfo 视频基础信息结构，供格式化函数使用。
type VideoInfo struct {
	ID         uint64   `json:"id"`
	Title      string   `json:"title"`
	CoverURL   string   `json:"cover_url"`
	AuthorID   uint64   `json:"author_id"`
	AuthorName string   `json:"author_name"`
	PlayCount  int64    `json:"play_count"`
	LikeCount  int64    `json:"like_count"`
	Duration   int      `json:"duration"`
	CreateTime string   `json:"create_time"`
	Tags       []string `json:"tags,omitempty"`
}

// RequestParams 来自前端的精确请求参数。
type RequestParams struct {
	VideoID  uint64 `json:"video_id,omitempty"`
	Keyword  string `json:"keyword,omitempty"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"page_size,omitempty"`
	UserID   uint64 `json:"user_id,omitempty"`
}

// ParseRequestParams 解析 query 中的请求参数前缀，返回纯 query 和解析后的参数。
// query 格式: ## 请求参数\nkey=value\n...\n\n## 用户查询\nactual_query
// 如果没有参数前缀，返回原始 query 和 nil。
func ParseRequestParams(query string) (string, *RequestParams) {
	const marker = "## 请求参数"
	idx := -1
	for i := 0; i < len(query) && i+len(marker) <= len(query); i++ {
		if query[i:i+len(marker)] == marker {
			idx = i
			break
		}
	}
	if idx == -1 {
		return query, nil
	}

	// 找到 ## 用户查询 标记，分隔参数块和实际 query
	const queryMarker = "## 用户查询"
	actualQueryStart := -1
	for i := idx + len(marker); i+len(queryMarker) <= len(query); i++ {
		if query[i:i+len(queryMarker)] == queryMarker {
			actualQueryStart = i + len(queryMarker)
			break
		}
	}

	var paramBlock string
	var actualQuery string
	if actualQueryStart != -1 {
		paramBlock = query[idx:actualQueryStart]
		actualQuery = query[actualQueryStart:]
	} else {
		paramBlock = query[idx:]
		actualQuery = ""
	}

	params := &RequestParams{}
	for _, line := range splitLines(paramBlock) {
		line = trimPrefix(line, "## 请求参数")
		line = trimPrefix(line, "## 用户查询")
		line = trimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		parts := split2(line, "=")
		if len(parts) != 2 {
			continue
		}
		key := trimSpace(parts[0])
		val := trimSpace(parts[1])
		switch key {
		case "video_id":
			params.VideoID = parseUint64(val)
		case "keyword":
			params.Keyword = val
		case "page":
			params.Page = int(parseUint64(val))
		case "page_size":
			params.PageSize = int(parseUint64(val))
		case "user_id":
			params.UserID = parseUint64(val)
		}
	}

	return trimSpace(actualQuery), params
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '\n' {
			line := s[start:i]
			start = i + 1
			lines = append(lines, line)
		}
	}
	return lines
}

func split2(s, sep string) []string {
	idx := -1
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			idx = i
			break
		}
	}
	if idx == -1 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+len(sep):]}
}

func trimSpace(s string) string {
	i, j := 0, len(s)-1
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r') {
		i++
	}
	for j >= i && (s[j] == ' ' || s[j] == '\t' || s[j] == '\r' || s[j] == '\n') {
		j--
	}
	return s[i : j+1]
}

func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func parseUint64(s string) uint64 {
	var n uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + uint64(c-'0')
	}
	return n
}

func BuildQueryWithParams(query string, videoID int64, keyword string, page, pageSize int, userID uint64) string {
	if videoID == 0 && keyword == "" && page == 0 && pageSize == 0 && userID == 0 {
		return query
	}
	var parts []string
	parts = append(parts, "## 请求参数")
	if videoID > 0 {
		parts = append(parts, "video_id="+int64ToString(videoID))
	}
	if keyword != "" {
		parts = append(parts, "keyword="+keyword)
	}
	if page > 0 {
		parts = append(parts, "page="+intToString(page))
	}
	if pageSize > 0 {
		parts = append(parts, "page_size="+intToString(pageSize))
	}
	if userID > 0 {
		parts = append(parts, "user_id="+uint64ToString(userID))
	}
	parts = append(parts, "")
	parts = append(parts, "## 用户查询")
	parts = append(parts, query)
	return strings.Join(parts, "\n")
}

func int64ToString(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func uint64ToString(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func intToString(n int) string {
	return int64ToString(int64(n))
}
