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

type FoodSearchParams struct {
	Query string `json:"query" jsonschema:"description=食品配料名称关键词"`
}

// 食品成分搜索处理函数
func foodSearchHandler(ctx context.Context, args FoodSearchParams) (*mcp.CallToolResult, error) {
	// 获取当前文件的绝对路径
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("获取可执行文件路径失败: %v", err)
	}

	// 获取项目根目录
	projectRoot := filepath.Dir(exePath)

	// 构建 food_digrents.txt 的绝对路径
	foodListPath := filepath.Join(projectRoot, "food_ingredients.txt")

	// 读取 food_digrents.txt 文件
	content, err := os.ReadFile(foodListPath)
	if err != nil {
		return nil, fmt.Errorf("读取食品成分文件失败: %v", err)
	}

	var markdownBuilder strings.Builder
	markdownBuilder.WriteString("## 食品成分搜索结果\n\n")
	markdownBuilder.WriteString(string(content))

	return mcp.NewToolResultText(markdownBuilder.String()), nil
}

var FoodSearchTool = hyancieMCP.MustTool(
	"food-ingredients-search",
	"食品成分搜索",
	foodSearchHandler,
)

func AddFoodSearchTools(mcp *server.MCPServer) {
	FoodSearchTool.Register(mcp)
}
