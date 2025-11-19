package logic

import (
	"context"
	"encoding/json"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	Estype "github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
	"xls/app/video/rpc/video/internal/code"

	"xls/app/video/rpc/video/internal/svc"
	"xls/app/video/rpc/video/video"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchVideoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSearchVideoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchVideoLogic {
	return &SearchVideoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SearchVideoLogic) SearchVideo(in *video.SearchVideoRequest) (*video.SearchVideoResponse, error) {
	resp := &video.SearchVideoResponse{}

	if in.Page <= 0 {
		in.Page = 1
	}
	if in.Size <= 0 {
		in.Size = 10
	}
	from := int((in.Page - 1) * in.Size)
	size := int(in.Size)
	order := sortorder.Desc

	esReq := &search.Request{
		Query: &Estype.Query{
			Bool: &Estype.BoolQuery{
				Must: []Estype.Query{
					{
						MultiMatch: &Estype.MultiMatchQuery{
							Query:  in.Keyword,
							Fields: []string{"title^3", "author_name"},
						},
					},
				},
				Filter: []Estype.Query{
					{
						Term: map[string]Estype.TermQuery{
							"deleted_at": {Value: ""},
						},
					},
				},
			},
		},
		From: ptrInt(from),
		Size: ptrInt(size),
		Sort: []Estype.SortCombinations{
			Estype.SortOptions{
				SortOptions: map[string]Estype.FieldSort{
					"like_num": {Order: &order},
				},
			},
			Estype.SortOptions{
				SortOptions: map[string]Estype.FieldSort{
					"created_at": {Order: &order},
				},
			},
		},
		Highlight: &Estype.Highlight{
			Fields: map[string]Estype.HighlightField{
				"title": {},
			},
			PreTags:  []string{"<em>"},
			PostTags: []string{"</em>"},
		},
	}

	result, err := l.svcCtx.TypedEs.Search().Index("video-index").Request(esReq).Do(l.ctx)
	if err != nil {
		l.Logger.Errorf("[SearchVideoLogic]SearchVideo err: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}

	resp.Total = result.Hits.Total.Value

	for _, hit := range result.Hits.Hits {
		var item *video.VideoItem
		if err := json.Unmarshal(hit.Source_, &item); err != nil {
			l.Logger.Errorf("[SearchVideoLogic]Unmarshal err: %v", err)
			continue
		}

		if h, ok := hit.Highlight["title"]; ok && len(h) > 0 {
			item.Title = h[0]
		}

		resp.Videos = append(resp.Videos, item)
	}

	resp.Error = code.SUCCEED

	return resp, nil
}

func ptrInt(i int) *int {
	return &i
}
