package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"xls/app/agent/api/internal/logic"
	"xls/app/agent/api/internal/svc"
	"xls/app/agent/api/internal/types"
)

func chatHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ChatReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewChatLogic(r.Context(), svcCtx)
		resp, err := l.Chat(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
