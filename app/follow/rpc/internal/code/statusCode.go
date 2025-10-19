package code

import (
	"xls/app/follow/rpc/follow"
)

var (
	SUCCEED           = response(0, "成功")
	FAILED            = response(1, "失败")
	FollowStatusError = response(40001, "关注状态不匹配")
)

func response(code int32, msg string) *follow.Error {
	err := new(follow.Error)
	err.Code = code
	err.Message = msg
	return err
}
