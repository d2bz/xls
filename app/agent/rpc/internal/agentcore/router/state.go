package router

import (
	"xls/app/agent/rpc/internal/agentcore/workflows"
)

// AgentState 是整个 Graph 的局部状态，每个 Graph.Invoke 调用创建一个新实例。
// 在 Lambda/Branch 中通过 compose.ProcessState[*AgentState] 访问。
type AgentState struct {
	Intent     workflows.Intent
	Slots      *workflows.TaskSlot
	Confidence float64
	Answer     string
	Error      string
	Reason     string
}
