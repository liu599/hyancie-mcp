package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	hyancieMCP "github.com/liu599/hyancie"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type IndustrySearchParams struct {
	Query string `json:"query" jsonschema:"description=搜索行业的关键词，留空则返回所有行业"`
}

// IndustrySearchTool 处理函数
func industrySearchHandler(ctx context.Context, args IndustrySearchParams) (*mcp.CallToolResult, error) {
	// 获取当前文件的绝对路径
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("获取可执行文件路径失败: %v", err)
	}

	// 获取项目根目录
	projectRoot := filepath.Dir(exePath)

	// 构建 industries.txt 的绝对路径
	industriesPath := filepath.Join(projectRoot, "industries.txt")

	// 读取 industries.txt 文件
	content, err := os.ReadFile(industriesPath)
	if err != nil {
		return nil, fmt.Errorf("读取行业列表文件失败: %v", err)
	}

	// 将文件内容按行分割
	industries := strings.Split(string(content), "\n")

	// 构建 markdown 格式的结果
	var markdownBuilder strings.Builder
	markdownBuilder.WriteString("## 支持的行业列表\n\n")

	// 如果没有搜索关键词，返回所有行业
	if args.Query == "" {
		for i, industry := range industries {
			if industry == "" {
				continue
			}
			markdownBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, industry))
		}
	} else {
		// 如果有搜索关键词，进行过滤
		found := false
		for i, industry := range industries {
			if strings.Contains(industry, args.Query) {
				found = true
				markdownBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, industry))
			}
		}
		if !found {
			markdownBuilder.WriteString("未找到匹配的行业\n")
		}
	}

	return mcp.NewToolResultText(markdownBuilder.String()), nil
}

var IndustrySearchTool = hyancieMCP.MustTool(
	"search-supported-industries",
	"搜索系统支持的行业",
	industrySearchHandler,
)

func AddIndustrySearchTools(mcp *server.MCPServer) {
	IndustrySearchTool.Register(mcp)
}
