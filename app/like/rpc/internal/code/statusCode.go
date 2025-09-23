package code

import "xls/app/like/rpc/like"

var (
	SUCCEED = response(0, "成功")
	FAILED  = response(1, "失败")
)

func response(code int32, msg string) *like.Error {
	err := new(like.Error)
	err.Code = code
	err.Message = msg
	return err
}
