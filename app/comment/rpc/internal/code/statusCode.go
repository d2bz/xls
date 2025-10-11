package code

import (
	"xls/app/comment/rpc/comment"
)

var (
	SUCCEED = response(0, "成功")
	FAILED  = response(1, "失败")
)

func response(code int32, msg string) *comment.Error {
	err := new(comment.Error)
	err.Code = code
	err.Message = msg
	return err
}
