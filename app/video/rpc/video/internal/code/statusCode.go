package code

import "xls/app/video/rpc/video/video"

var (
	SUCCEED = response(0, "成功")
	FAILED  = response(1, "失败")
)

func response(code int32, msg string) *video.Error {
	err := new(video.Error)
	err.Code = code
	err.Message = msg
	return err
}
