package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"xls/app/agent/api/internal/svc"
	"xls/app/agent/api/internal/types"
	"xls/app/agent/rpc/agentclient"
)

type ChatLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatLogic {
	return &ChatLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChatLogic) Chat(req *types.ChatReq) (*types.ChatResp, error) {
	resp := &types.ChatResp{}

	uid, _ := l.ctx.Value("user_id").(json.Number).Int64()

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("session_%d", uid)
	}

	result, err := l.svcCtx.AgentRpc.Chat(l.ctx, &agentclient.ChatRequest{
		UserId:    uint64(uid),
		Query:     req.Query,
		SessionId: sessionID,
	})
	if err != nil {
		log.Printf("[ERROR] call agent rpc error: %v", err)
		resp.StatusCode = 500
		resp.StatusMsg = err.Error()
		resp.Answer = "抱歉，服务出现了问题：" + err.Error()
		resp.SessionID = sessionID
		return resp, nil
	}

	resp.StatusCode = int(result.StatusCode)
	resp.StatusMsg = result.StatusMsg
	resp.Answer = result.Answer
	resp.SessionID = result.SessionId

	return resp, nil
}
