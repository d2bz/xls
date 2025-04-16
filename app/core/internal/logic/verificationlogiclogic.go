package logic

import (
	"context"
	"strings"

	"xls/app/core/internal/code"
	"xls/app/core/internal/helper"
	"xls/app/core/internal/svc"
	"xls/app/core/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type VerificationLogicLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewVerificationLogicLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerificationLogicLogic {
	return &VerificationLogicLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VerificationLogicLogic) VerificationLogic(req *types.VerificationRequest) (resp *types.VerificationResponse, err error) {
	resp = new(types.VerificationResponse)
	req.Email = strings.TrimSpace(req.Email)
	matched := helper.CheckEmail(req.Email)
	if !matched {
		resp.Status = code.EmailFormatErorr
		return
	}
	return
}
