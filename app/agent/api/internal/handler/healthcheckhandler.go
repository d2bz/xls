package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"xls/app/agent/api/internal/logic"
	"xls/app/agent/api/internal/svc"
	"xls/app/agent/api/internal/types"
)

// 服务健康检查
func healthCheckHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.HealthCheckReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewHealthCheckLogic(r.Context(), svcCtx)
		resp, err := l.HealthCheck(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
