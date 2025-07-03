package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	hyancieMCP "github.com/liu599/hyancie"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type JinaSearchParams struct {
	URL string `json:"url" jsonschema:"required,description=要搜索的HTML地址"`
}

// JinaSearchTool 处理函数
func jinaSearchHandler(ctx context.Context, args JinaSearchParams) (*mcp.CallToolResult, error) {
	// 构建Jina API请求URL
	jinaURL := fmt.Sprintf("https://r.jina.ai/%s", args.URL)

	// 创建请求
	req, err := http.NewRequest("GET", jinaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建Jina请求失败: %v", err)
	}

	// 添加认证头
	req.Header.Add("Authorization", "Bearer jina_407a2f1dab98423d81580d69b6b9588c7bKW6aajEjm36klnhqG8Vl4jaI0r")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Jina请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Jina API返回错误状态码: %d", resp.StatusCode)
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取Jina响应失败: %v", err)
	}

	// 构建结果
	var markdownBuilder strings.Builder
	markdownBuilder.WriteString("## Jina搜索结果\n\n")
	markdownBuilder.WriteString(string(body))

	return mcp.NewToolResultText(markdownBuilder.String()), nil
}

var JinaSearchTool = hyancieMCP.MustTool(
	"jina-search",
	"通过Jina AI获取网页内容",
	jinaSearchHandler,
)

func AddJinaSearchTools(mcp *server.MCPServer) {
	JinaSearchTool.Register(mcp)
}
