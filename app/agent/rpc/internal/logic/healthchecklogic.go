package logic

import (
	"context"

	"xls/app/agent/rpc/agent"
	"xls/app/agent/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type HealthCheckLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewHealthCheckLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HealthCheckLogic {
	return &HealthCheckLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 健康检查
func (l *HealthCheckLogic) HealthCheck(in *agent.HealthCheckRequest) (*agent.HealthCheckResponse, error) {
	// todo: add your logic here and delete this line

	return &agent.HealthCheckResponse{}, nil
}
