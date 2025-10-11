package logic

import (
	"context"
	"xls/app/comment/rpc/internal/code"
	"xls/app/comment/rpc/internal/model"

	"xls/app/comment/rpc/comment"
	"xls/app/comment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CommentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CommentLogic {
	return &CommentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CommentLogic) Comment(in *comment.CommentRequest) (*comment.CommentResponse, error) {
	// todo: add your logic here and delete this line
	resp := new(comment.CommentResponse)

	commentMsg := &model.Comment{
		UserID:       in.UserID,
		TargetID:     in.TargetID,
		TargetUserID: in.TargetUserID,
		ParentID:     in.ParentID,
		Content:      in.Content,
	}

	if err := commentMsg.InsertComment(l.svcCtx.MysqlDB); err != nil {
		l.Logger.Errorf("Insert comment error: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}

	resp.Error = code.SUCCEED

	return resp, nil
}
