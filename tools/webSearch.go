package tools

import (
	"context"
	"encoding/json"
	"fmt"
	hyancieMCP "github.com/liu599/hyancie"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type WebSearchParams struct {
	Query string `json:"query" jsonschema:"required,description=搜索查询字符串"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

type SearchResult struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// WebSearchTool 处理函数
func webSearchHandler(ctx context.Context, args WebSearchParams) (*mcp.CallToolResult, error) {
	// 从配置中获取搜索API的URL
	searchURL := hyancieMCP.Config.WebSearchURL
	if searchURL == "" {
		return nil, fmt.Errorf("未配置搜索API URL")
	}

	// 构建请求URL
	baseURL, err := url.Parse(searchURL)
	if err != nil {
		return nil, fmt.Errorf("解析搜索URL失败: %v", err)
	}

	params := url.Values{}
	params.Add("q", args.Query)
	params.Add("format", "json")
	baseURL.RawQuery = params.Encode()

	// 创建新的请求
	req, err := http.NewRequest("GET", baseURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 添加API Key到请求头
	req.Header.Add("X-API-Key", hyancieMCP.Config.APIKey)

	// 发送HTTP请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("搜索请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析JSON响应
	var searchResponse SearchResponse
	if err := json.Unmarshal(body, &searchResponse); err != nil {
		return nil, fmt.Errorf("解析搜索结果失败: %v\n响应体内容: %s", err, string(body))
	}

	// 格式化结果
	var markdownBuilder strings.Builder
	markdownBuilder.WriteString("## 搜索结果\n\n")

	if len(searchResponse.Results) == 0 {
		markdownBuilder.WriteString("没有搜索结果, 请参考其他信息\n")
	} else {
		// 只取前10个结果
		results := searchResponse.Results
		if len(results) > 3 {
			results = results[:3]
		}

		for i, result := range results {
			// 跳过URL为空的结果
			if result.URL == "" {
				continue
			}

			markdownBuilder.WriteString(fmt.Sprintf("### 第%d篇\n", i+1))
			markdownBuilder.WriteString(fmt.Sprintf("- 标题: %s\n", result.Title))
			markdownBuilder.WriteString(fmt.Sprintf("- 简介: %s\n", result.Content))
			markdownBuilder.WriteString(fmt.Sprintf("- 链接: %s\n\n", result.URL))
		}
	}

	return mcp.NewToolResultText(markdownBuilder.String()), nil
}

var WebSearchTool = hyancieMCP.MustTool(
	"food-info-search",
	"食品网页搜索工具",
	webSearchHandler,
)

func AddWebSearchTools(mcp *server.MCPServer) {
	WebSearchTool.Register(mcp)
}
