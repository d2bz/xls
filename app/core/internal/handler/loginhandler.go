package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"xls/app/core/internal/logic"
	"xls/app/core/internal/svc"
	"xls/app/core/internal/types"
)

func LoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LoginRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewLoginLogicLogic(r.Context(), svcCtx)
		resp, err := l.LoginLogic(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
