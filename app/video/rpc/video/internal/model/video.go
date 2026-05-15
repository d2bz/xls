package model

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type Video struct {
	Uid   uint   `gorm:"not null" json:"uid"`
	Title string `gorm:"varchar(255);not null" json:"title"`
	Url   string `gorm:"varchar(255); not null" json:"url"`

	Tags []Tag `gorm:"many2many:video_tags;" json:"tags"`

	LikeNum    int `json:"like_num"`
	CommentNum int `json:"comment_num"`
	gorm.Model
}

func (v *Video) TableName() string {
	return "video"
}

func (v *Video) Insert(db *gorm.DB) error {
	return db.Create(v).Error
}

func FindVideoByID(db *gorm.DB, videoID uint) (*Video, error) {
	var video *Video
	err := db.Preload("Tags").Where("id = ?", videoID).First(video).Error
	return video, err
}

func FindVideosByIDs(db *gorm.DB, ids []uint) ([]*Video, error) {
	if len(ids) == 0 {
		return []*Video{}, nil
	}
	var videos []*Video
	if err := db.Preload("Tags").Where("id IN ?", ids).Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}

func FindVideosByTag(db *gorm.DB, tag string, offset, limit int) ([]*Video, int64, error) {
	var videos []*Video
	var total int64

	if err := db.Model(&Video{}).Where("id IN (?)", db.Table("video_tags").Select("video_id").Joins("JOIN tag ON tag.id = video_tags.tag_id").Where("tag.name = ?", tag)).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Preload("Tags").
		Where("id IN (?)", db.Table("video_tags").Select("video_id").Joins("JOIN tag ON tag.id = video_tags.tag_id").Where("tag.name = ?", tag)).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&videos).Error; err != nil {
		return nil, 0, err
	}

	return videos, total, nil
}

type DimensionInput struct {
	Name   string
	Tags   []string
	Weight int
}

func FindVideosByDimensions(db *gorm.DB, dimensions []DimensionInput, limit, offset int) ([]*Video, int64, error) {
	if len(dimensions) == 0 {
		return nil, 0, nil
	}

	// 构建 CASE 打分 SQL，按加权分数排序
	// 公式: SUM(1.0 / weight) for matched dims, weight 越小越重要
	var caseScore strings.Builder
	var caseArgs []any
	for i, dim := range dimensions {
		if len(dim.Tags) == 0 {
			continue
		}
		if i > 0 {
			caseScore.WriteString(" + ")
		}
		caseScore.WriteString(fmt.Sprintf(
			"CASE WHEN EXISTS (SELECT 1 FROM video_tags vt%di JOIN tag t%di ON t%di.id = vt%di.tag_id WHERE vt%di.video_id = v.id AND t%di.name IN (",
			i, i, i, i, i, i,
		))
		placeholders := make([]string, len(dim.Tags))
		for j := range dim.Tags {
			placeholders[j] = "?"
			caseArgs = append(caseArgs, dim.Tags[j])
		}
		caseScore.WriteString(strings.Join(placeholders, ","))
		caseScore.WriteString(fmt.Sprintf(")) THEN %.6f / %d ELSE 0 END", 1.0, dim.Weight))
	}

	// 检查是否全维度无 tags
	if caseScore.Len() == 0 {
		return nil, 0, nil
	}

	scoreExpr := caseScore.String()

	// 构建 OR 过滤条件：至少有一个维度的标签匹配
	var existClauses []string
	var existArgs []any
	for i, dim := range dimensions {
		if len(dim.Tags) == 0 {
			continue
		}
		placeholders := make([]string, len(dim.Tags))
		for j := range dim.Tags {
			placeholders[j] = "?"
			existArgs = append(existArgs, dim.Tags[j])
		}
		existClauses = append(existClauses, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM video_tags vt%di JOIN tag t%di ON t%di.id = vt%di.tag_id WHERE vt%di.video_id = v.id AND t%di.name IN (%s))",
			i, i, i, i, i, i, strings.Join(placeholders, ","),
		))
	}
	whereClause := strings.Join(existClauses, " OR ")

	// 最小匹配分数：1.0 / 最大权重，保证至少匹配一个维度
	// weight 越小越重要（权重越高），0 weight 视为最高优先级，默认用 1
	maxWeight := 1
	for _, dim := range dimensions {
		if dim.Weight > maxWeight {
			maxWeight = dim.Weight
		}
	}
	minScore := 1.0 / float64(maxWeight)

	// 计数（部分匹配，按分数过滤）
	var total int64
	countSQL := fmt.Sprintf(`
		SELECT COUNT(*) FROM (
			SELECT v.id, %s AS score
			FROM video v
			WHERE %s
			GROUP BY v.id
			HAVING score >= %f
		) AS t`,
		scoreExpr, whereClause, minScore)
	if err := db.Raw(countSQL, existArgs...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询（按分数降序，支持部分匹配）
	querySQL := fmt.Sprintf(`
		SELECT v.id, %s AS score
		FROM video v
		WHERE %s
		GROUP BY v.id
		HAVING score >= %f
		ORDER BY score DESC
		LIMIT ? OFFSET ?`,
		scoreExpr, whereClause, minScore)

	type videoScore struct {
		ID    uint `gorm:"primaryKey"`
		Score float64
	}
	var results []videoScore
	queryArgs := append(caseArgs, limit, offset)
	if err := db.Raw(querySQL, queryArgs...).Scan(&results).Error; err != nil {
		return nil, 0, err
	}

	if len(results) == 0 {
		return nil, int64(0), nil
	}

	// 按原始顺序返回
	ids := make([]uint, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}

	var videos []*Video
	if err := db.Preload("Tags").
		Where("id IN ?", ids).
		Find(&videos).Error; err != nil {
		return nil, 0, err
	}

	// 保持分数降序
	idToVideo := make(map[uint]*Video, len(videos))
	for _, v := range videos {
		idToVideo[v.ID] = v
	}
	ordered := make([]*Video, 0, len(ids))
	for _, id := range ids {
		if v, ok := idToVideo[id]; ok {
			ordered = append(ordered, v)
		}
	}

	return ordered, total, nil
}
