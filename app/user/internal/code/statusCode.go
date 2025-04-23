package code

import "xls/app/user/user"

var (
	UserIsExist = response(20001, "用户已存在")
)

func response(code int32, msg string) *user.Error {
	err := new(user.Error)
	err.Code = code
	err.Message = msg
	return err
}
