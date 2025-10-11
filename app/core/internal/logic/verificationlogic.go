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

type VerificationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewVerificationLogicLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerificationLogic {
	return &VerificationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VerificationLogic) VerificationLogic(req *types.VerificationRequest) (resp *types.VerificationResponse, err error) {
	resp = new(types.VerificationResponse)
	req.Email = strings.TrimSpace(req.Email)

	// 验证输入格式
	matched := helper.CheckEmailFormat(req.Email)
	if !matched {
		resp.Status = code.EmailFormatErorr
		return resp, nil
	}

	// 检查验证码冷却时间
	cd, err := l.svcCtx.BizRedis.Get(prefixCD + req.Email)
	if err != nil {
		l.Logger.Errorf("get verification code cd failed: %v", err)
		resp.Status = code.FAILED
		return resp, nil
	} else if cd == "1" {
		resp.Status = code.VerificationCodeIsCoolDown
		return resp, nil
	}

	// 生成验证码
	vcode, err := helper.GenRandomCode(code_length)
	if err != nil {
		l.Logger.Errorf("generate random code failed: %v", err)
		resp.Status = code.FAILED
		return resp, nil
	}

	// 发送验证码
	err = send_email.SendEmail(req.Email, vcode)
	if err != nil {
		l.Logger.Errorf("send email failed: %v", err)
		resp.Status = code.FAILED
		return resp, nil
	}

	// 缓存验证码
	err = l.saveCodeToRedis(req.Email, vcode)
	if err != nil {
		l.Logger.Errorf("set verification code cache failed: %v", err)
		resp.Status = code.FAILED
		return resp, nil
	}
	return
}

func (l *VerificationLogic) saveCodeToRedis(email, code string) (err error) {
	vcKey := prefixVC + email
	err = l.svcCtx.BizRedis.Setex(vcKey, code, expireVC)
	if err != nil {
		return
	}
	cdKey := prefixCD + email
	err = l.svcCtx.BizRedis.Setex(cdKey, "1", expireCD)
	return
}
