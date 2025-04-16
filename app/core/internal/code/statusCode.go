package code

import "xls/app/core/internal/types"

var (
	SUCCEED          = response(0, "成功")
	EmailFormatErorr = response(10001, "邮箱格式错误")
)

func response(code int, msg string) types.Status {
	resp := new(types.Status)
	resp.StatusCode = code
	resp.StatusMsg = msg
	return *resp
}
