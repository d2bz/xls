package code

import "xls/app/core/internal/types"

var (
	SUCCEED                    = response(0, "成功")
	FAILED                     = response(1, "失败")
	EmailFormatErorr           = response(10001, "邮箱格式错误")
	VerificationCodeIsCoolDown = response(10002, "验证码冷却中")
	WrongVerificationCode      = response(10003, "验证码错误")
	VerificationCodeIsEmpty    = response(10004, "验证码为空")
	PasswordFormatError        = response(10005, "密码格式错误")
	NoLogin                    = response(10006, "未登录")
)

func response(code int, msg string) types.Status {
	resp := new(types.Status)
	resp.StatusCode = code
	resp.StatusMsg = msg
	return *resp
}
