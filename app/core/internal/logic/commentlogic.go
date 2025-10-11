package logic

import (
	"context"
	"encoding/json"
	"xls/app/comment/rpc/comment"
	"xls/app/core/internal/code"

	"xls/app/core/internal/svc"
	"xls/app/core/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CommentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CommentLogic {
	return &CommentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CommentLogic) Comment(req *types.CommentRequest) (resp *types.CommentResponse, err error) {
	resp = new(types.CommentResponse)
	uid, err := l.ctx.Value("userid").(json.Number).Int64()
	if err != nil {
		resp.Status = code.NoLogin
		return resp, nil
	}

	commentResp, err := l.svcCtx.CommentRpc.Comment(l.ctx, &comment.CommentRequest{
		UserID:       uint64(uid),
		TargetID:     req.TargetID,
		TargetUserID: req.TargetUserID,
		ParentID:     req.ParentID,
		Content:      req.Content,
	})
	if err != nil {
		resp.Status = code.FAILED
		l.Logger.Errorf("comment err: %v", err)
		return resp, nil
	}
	if commentResp.Error.Code != 0 {
		resp.Status.StatusCode = int(commentResp.Error.Code)
		resp.Status.StatusMsg = commentResp.Error.Message
		return resp, nil
	}

	resp.Status = code.SUCCEED

	return resp, nil
}
