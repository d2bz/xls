package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"xls/app/core/internal/logic"
	"xls/app/core/internal/svc"
	"xls/app/core/internal/types"
)

func RegisterLogicHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RegisterRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewRegisterLogicLogic(r.Context(), svcCtx)
		resp, err := l.RegisterLogic(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
