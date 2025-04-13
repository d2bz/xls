package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"xls/app/core/internal/logic"
	"xls/app/core/internal/svc"
	"xls/app/core/internal/types"
)

func CoreHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.Request
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewCoreLogic(r.Context(), svcCtx)
		resp, err := l.Core(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
