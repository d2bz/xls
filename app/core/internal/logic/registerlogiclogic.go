package logic

import (
	"context"
	"strings"

	"xls/app/core/internal/code"
	"xls/app/core/internal/helper"
	"xls/app/core/internal/svc"
	"xls/app/core/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type RegisterLogicLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogicLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogicLogic {
	return &RegisterLogicLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogicLogic) RegisterLogic(req *types.RegisterRequest) (resp *types.RegisterResponse, err error) {
	resp = new(types.RegisterResponse)
	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	req.VerificationCode = strings.TrimSpace(req.VerificationCode)
	matched := helper.CheckEmail(req.Email)
	if !matched {
		resp.Status = code.EmailFormatErorr
		return
	}
	matched = helper.CheckPassword(req.Password)
	if !matched {
		resp.Status = code.PasswordFormatError
		return
	}

	rdbCode, err := l.svcCtx.BizRedis.Get(prefixVC + req.Email)
	if err == redis.Nil {
		resp.Status = code.VerificationCodeIsEmpty
		return
	} else if err != nil {
		logx.Errorf("get verification code cd failed: %v", err)
		resp.Status = code.FAILED
		return
	} else if rdbCode != req.VerificationCode {
		resp.Status = code.WrongVerificationCode
		return
	}
	return
}
