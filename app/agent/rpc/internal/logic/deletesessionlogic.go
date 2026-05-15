package logic

import (
	"context"
	"errors"

	"xls/app/agent/rpc/agent"
	"xls/app/agent/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type DeleteSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSessionLogic {
	return &DeleteSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteSessionLogic) DeleteSession(in *agent.DeleteSessionRequest) (*agent.DeleteSessionResponse, error) {
	resp := &agent.DeleteSessionResponse{}

	if l.svcCtx.SessionStore == nil {
		resp.StatusCode = 500
		resp.StatusMsg = "session store not initialized"
		return resp, nil
	}

	if in.SessionUuid == "" {
		resp.StatusCode = 400
		resp.StatusMsg = "session_uuid is required"
		return resp, nil
	}

	err := l.svcCtx.SessionStore.Delete(l.ctx, in.SessionUuid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			resp.StatusCode = 404
			resp.StatusMsg = "session not found"
			return resp, nil
		}
		logx.Errorf("[DeleteSession] delete error: %v", err)
		resp.StatusCode = 500
		resp.StatusMsg = err.Error()
		return resp, nil
	}

	resp.StatusCode = 0
	resp.StatusMsg = "success"
	return resp, nil
}
