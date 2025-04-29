package code

import "xls/app/user/user"

var (
	SUCCEED           = response(0, "成功")
	FAILED            = response(1, "失败")
	UserAlreadyExists = response(20001, "用户已存在")
	UserNotFound      = response(20002, "用户不存在")
)

func response(code int32, msg string) *user.Error {
	err := new(user.Error)
	err.Code = code
	err.Message = msg
	return err
}
