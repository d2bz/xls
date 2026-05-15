package mcpclient

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	mcproto "github.com/mark3labs/mcp-go/mcp"
	"xls/app/agent/rpc/internal/config"
)

// Client wraps an MCP client and its tools.
type Client struct {
	cli   *client.Client
	tools []tool.BaseTool
}

// NewClient creates an MCP client and initializes the connection.
// For stdio transport, starts the MCP server as a subprocess.
// For SSE transport, connects to an already-running SSE server.
func NewClient(ctx context.Context, cfg config.VideoMCPConfig) (*Client, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	var cli *client.Client
	var err error

	switch cfg.Transport {
	case "sse", "SSE":
		if cfg.Address == "" {
			return nil, fmt.Errorf("MCP SSE address is required when transport is 'sse'")
		}
		cli, err = client.NewSSEMCPClient(cfg.Address)
		if err != nil {
			return nil, fmt.Errorf("create SSE MCP client: %w", err)
		}
		if err = cli.Start(ctx); err != nil {
			return nil, fmt.Errorf("start SSE MCP client: %w", err)
		}
	default:
		// stdio: 启动 Python MCP 服务器作为子进程
		// NewStdioMCPClient 会自动 Start，不需要手动调用
		args := []string{"-m", "aigroup_video_mcp.main", "serve"}
		cli, err = client.NewStdioMCPClient("python", nil, args...)
		if err != nil {
			return nil, fmt.Errorf("create stdio MCP client: %w", err)
		}
	}

	// Initialize
	initReq := mcproto.InitializeRequest{
		Params: mcproto.InitializeParams{
			ProtocolVersion: mcproto.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcproto.Implementation{
				Name:    "xls-agent",
				Version: "1.0.0",
			},
		},
	}

	_, err = cli.Initialize(ctx, initReq)
	if err != nil {
		return nil, fmt.Errorf("MCP initialize: %w", err)
	}

	// Get tools
	mcpCfg := &mcp.Config{
		Cli: cli,
	}
	if len(cfg.ToolFilters) > 0 {
		mcpCfg.ToolNameList = cfg.ToolFilters
	}

	tools, err := mcp.GetTools(ctx, mcpCfg)
	if err != nil {
		return nil, fmt.Errorf("get MCP tools: %w", err)
	}

	return &Client{cli: cli, tools: tools}, nil
}

// Tools returns the list of MCP tools.
func (c *Client) Tools() []tool.BaseTool {
	if c == nil {
		return nil
	}
	return c.tools
}

// Close closes the MCP client connection.
func (c *Client) Close(ctx context.Context) {
	if c == nil || c.cli == nil {
		return
	}
	_ = c.cli.Close()
}

// MustNewClient is like NewClient but returns nil on error (non-fatal).
func MustNewClient(ctx context.Context, cfg config.VideoMCPConfig) *Client {
	c, err := NewClient(ctx, cfg)
	if err != nil {
		if cfg.Enabled {
			fmt.Fprintf(os.Stderr, "[WARN] MCP client init failed: %v, continuing without MCP tools\n", err)
		}
		return nil
	}
	return c
}
