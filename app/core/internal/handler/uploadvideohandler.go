package handler

import (
	"net/http"

	"xls/app/core/internal/logic"
	"xls/app/core/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func UploadVideoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileData, header, err := r.FormFile("file")
		if err != nil {
			httpx.Error(w, err)
			return
		}

		l := logic.NewUploadVideoLogic(r.Context(), svcCtx)
		resp, err := l.UploadVideo(&fileData, header)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}

		// size := header.Size
		// const maxSize = 20 << 20

		// l := logic.NewUploadVideoLogic(r.Context(), svcCtx)

		// if size < maxSize {
		// 	var buf bytes.Buffer
		// 	_, err = io.Copy(&buf, fileData)
		// 	if err != nil {
		// 		httpx.Error(w, err)
		// 		return
		// 	}
		// 	resp, err := l.UploadSmallVideo(buf.Bytes(), header.Filename)
		// 	if err != nil {
		// 		httpx.ErrorCtx(r.Context(), w, err)
		// 	} else {
		// 		httpx.OkJsonCtx(r.Context(), w, resp)
		// 	}
		// } else {
		// 	resp, err := l.UploadLargeVideo(&fileData, header.Filename)
		// 	if err != nil {
		// 		httpx.ErrorCtx(r.Context(), w, err)
		// 	} else {
		// 		httpx.OkJsonCtx(r.Context(), w, resp)
		// 	}
		// }

	}
}
