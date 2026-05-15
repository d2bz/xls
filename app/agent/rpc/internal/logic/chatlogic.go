package logic

import (
	"context"
	"fmt"
	"strings"

	"xls/app/agent/rpc/agent"
	"xls/app/agent/rpc/internal/agentcore"
	"xls/app/agent/rpc/internal/agentcore/memory"
	"xls/app/agent/rpc/internal/svc"

	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
)

type ChatLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatLogic {
	return &ChatLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ChatLogic) Chat(in *agent.ChatRequest) (*agent.ChatResponse, error) {
	resp := &agent.ChatResponse{
		SessionId: in.SessionId,
	}

	if l.svcCtx.AgentCore == nil || l.svcCtx.AgentCore.Runner == nil {
		resp.StatusCode = 500
		resp.StatusMsg = "agent not initialized"
		resp.Answer = "抱歉，AI助理服务暂未启用，请联系管理员"
		return resp, nil
	}

	store := l.svcCtx.SessionStore
	runner := l.svcCtx.AgentCore.Runner

	// 1. 获取或创建 Session
	session, err := store.GetOrCreate(l.ctx, in.UserId, in.SessionId)
	if err != nil {
		logx.Errorf("[Chat] get or create session error: %v", err)
		resp.StatusCode = 500
		resp.StatusMsg = err.Error()
		resp.Answer = "抱歉，创建会话失败：" + err.Error()
		return resp, nil
	}

	// 2. 注入请求参数到 Query 字符串前缀
	// Supervisor 会解析前缀并存入 State，后续节点通过 ProcessState 读取
	queryWithParams := agentcore.BuildQueryWithParams(in.Query, in.VideoId, in.Keyword, in.Page, in.PageSize, in.UserId)

	// 3. 追加用户消息到 Session（先写入，保证持久化）
	userMsg := schema.UserMessage(queryWithParams)
	if err := store.Append(l.ctx, session.ID, userMsg); err != nil {
		logx.Errorf("[Chat] append user message error: %v", err)
	}

	// 4. 读取完整历史消息
	history, err := store.GetMessages(l.ctx, session.ID)
	if err != nil {
		logx.Errorf("[Chat] get messages error: %v", err)
		resp.StatusCode = 500
		resp.StatusMsg = err.Error()
		resp.Answer = "抱歉，读取会话历史失败：" + err.Error()
		resp.SessionId = session.UUID
		return resp, nil
	}
	logx.Infof("[Chat] session=%s, history_count=%d, query=%s",
		session.UUID, len(history), in.Query)

	// 5. 截断过长历史，控制 token 消耗
	maxHistory := 50
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
	}

	// 6. 调用 Agent（多轮）
	events := runner.Run(l.ctx, history)
	answer, err := agentcore.ExtractTextFromEvents(events)
	if err != nil {
		logx.Errorf("[Chat] runner run error: %v", err)
		resp.StatusCode = 500
		resp.StatusMsg = err.Error()
		resp.Answer = "抱歉，服务出现了问题：" + err.Error()
		resp.SessionId = session.UUID
		return resp, nil
	}

	// 7. 追加助手回复到 Session
	assistantMsg := schema.AssistantMessage(answer, nil)
	if err := store.Append(l.ctx, session.ID, assistantMsg); err != nil {
		logx.Errorf("[Chat] append assistant message error: %v", err)
	}

	resp.StatusCode = 0
	resp.StatusMsg = "success"
	resp.Answer = answer
	resp.SessionId = session.UUID

	// 8. 填充结构化数据
	agentcore.FillStructuredResponse(session.UUID, store, resp)

	return resp, nil
}

// ListSessions 列出用户的所有会话摘要。
func (l *ChatLogic) ListSessions(userID uint64) ([]memory.SessionMeta, error) {
	return l.svcCtx.SessionStore.List(l.ctx, userID)
}

// DeleteSession 删除指定会话。
func (l *ChatLogic) DeleteSession(sessionUUID string) error {
	return l.svcCtx.SessionStore.Delete(l.ctx, sessionUUID)
}
