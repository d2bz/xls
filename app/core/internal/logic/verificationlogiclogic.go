package logic

import (
	"context"
	"strings"

	"xls/app/core/internal/code"
	"xls/app/core/internal/helper"
	"xls/app/core/internal/svc"
	"xls/app/core/internal/types"
	"xls/pkg/send_email"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	code_length = 6
	prefixVC    = "verification#"
	prefixCD    = "verfication#cd#"
	expireVC    = 60 * 3 // 验证码过期时间
	expireCD    = 60 * 1 // 发送验证码间隔
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

	cd, err := l.svcCtx.BizRedis.Get(prefixCD + req.Email)
	if err != nil {
		logx.Errorf("get verification code cd failed: %v", err)
		resp.Status = code.FAILED
		return
	} else if cd == "1" {
		resp.Status = code.VerificationCodeIsCoolDown
		return
	}

	vcode, err := helper.GenRandomCode(code_length)
	if err != nil {
		logx.Errorf("generate random code failed: %v", err)
		resp.Status = code.FAILED
		return
	}
	err = send_email.SendEmail(req.Email, vcode)
	if err != nil {
		logx.Errorf("send email failed: %v", err)
		resp.Status = code.FAILED
		return
	}

	err = l.saveCodeToRedis(req.Email, vcode)
	if err != nil {
		logx.Errorf("set verification code cache failed: %v", err)
		resp.Status = code.FAILED
		return
	}

	return
}

func (l *VerificationLogicLogic) saveCodeToRedis(email, code string) (err error) {
	vcKey := prefixVC + email
	err = l.svcCtx.BizRedis.Setex(vcKey, code, expireVC)
	if err != nil {
		return
	}
	cdKey := prefixCD + email
	err = l.svcCtx.BizRedis.Setex(cdKey, "1", expireCD)
	return
}
