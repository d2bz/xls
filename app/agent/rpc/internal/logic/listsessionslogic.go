package logic

import (
	"context"

	"xls/app/agent/rpc/agent"
	"xls/app/agent/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListSessionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSessionsLogic {
	return &ListSessionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListSessionsLogic) ListSessions(in *agent.ListSessionsRequest) (*agent.ListSessionsResponse, error) {
	resp := &agent.ListSessionsResponse{}

	if l.svcCtx.SessionStore == nil {
		resp.StatusCode = 500
		resp.StatusMsg = "session store not initialized"
		return resp, nil
	}

	metas, err := l.svcCtx.SessionStore.List(l.ctx, in.UserId)
	if err != nil {
		logx.Errorf("[ListSessions] list error: %v", err)
		resp.StatusCode = 500
		resp.StatusMsg = err.Error()
		return resp, nil
	}

	sessions := make([]*agent.SessionMeta, 0, len(metas))
	for _, m := range metas {
		sessions = append(sessions, &agent.SessionMeta{
			Id:        uint64(m.ID),
			Uuid:      m.UUID,
			Title:     m.Title,
			UserId:    m.UserID,
			CreatedAt: m.CreatedAt.Unix(),
		})
	}

	resp.StatusCode = 0
	resp.StatusMsg = "success"
	resp.Sessions = sessions
	return resp, nil
}
