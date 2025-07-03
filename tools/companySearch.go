package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	hyancieMCP "github.com/liu599/hyancie"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Company struct {
	SearchTag   string `json:"search_tag"`
	ChineseName string `json:"chinese_name"`
	EnglishName string `json:"english_name"`
	Revenue     string `json:"revenue"`
	Rank        int    `json:"rank"`
	Details     struct {
		Revenue      struct{ Value, Growth string } `json:"revenue"`
		Profit       struct{ Value, Growth string } `json:"profit"`
		Assets       struct{ Value, Growth string } `json:"assets"`
		Equity       struct{ Value, Growth string } `json:"equity"`
		ProfitMargin struct{ Value, Growth string } `json:"profitMargin"`
		ROA          struct{ Value, Growth string } `json:"roa"`
	} `json:"details"`
}

type CompanySearchParams struct {
	Type  string `json:"type" jsonschema:"enum=industry|company|country,description=搜索类型：行业/公司/国家"`
	Query string `json:"query" jsonschema:"description=搜索关键词"`
}

// CompanySearchTool 处理函数
func companySearchHandler(ctx context.Context, args CompanySearchParams) (*mcp.CallToolResult, error) {
	// 获取当前文件的绝对路径
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("获取可执行文件路径失败: %v", err)
	}

	// 获取项目根目录
	projectRoot := filepath.Dir(exePath)

	// 构建 company_list.json 的绝对路径
	companyListPath := filepath.Join(projectRoot, "company_list.json")

	// 读取 company_list.json 文件
	content, err := os.ReadFile(companyListPath)
	if err != nil {
		return nil, fmt.Errorf("读取公司列表文件失败: %v", err)
	}

	var companies []Company
	if err := json.Unmarshal(content, &companies); err != nil {
		return nil, fmt.Errorf("解析公司数据失败: %v", err)
	}

	var markdownBuilder strings.Builder
	markdownBuilder.WriteString("## 2024年500强公司搜索结果\n\n")

	switch args.Type {
	case "industry":
		// 按行业搜索
		var filteredCompanies []Company
		for _, company := range companies {
			if strings.Contains(company.SearchTag, args.Query) {
				filteredCompanies = append(filteredCompanies, company)
			}
		}
		sort.Slice(filteredCompanies, func(i, j int) bool {
			return filteredCompanies[i].Rank < filteredCompanies[j].Rank
		})

		if len(filteredCompanies) == 0 {
			markdownBuilder.WriteString("未找到该行业的公司\n")
		} else {
			markdownBuilder.WriteString(fmt.Sprintf("### %s行业公司列表（按排名排序）\n\n", args.Query))
			for _, company := range filteredCompanies {
				markdownBuilder.WriteString(fmt.Sprintf("#### 第%d名: %s\n", company.Rank, company.ChineseName))
				markdownBuilder.WriteString(fmt.Sprintf("- 英文名: %s\n", company.EnglishName))
				markdownBuilder.WriteString(fmt.Sprintf("- 营收: %s百万美元\n", company.Revenue))
				markdownBuilder.WriteString(fmt.Sprintf("- 利润: %s百万美元 (增长率: %s)\n", company.Details.Profit.Value, company.Details.Profit.Growth))
				markdownBuilder.WriteString(fmt.Sprintf("- 利润率: %s\n", company.Details.ProfitMargin.Value))
				markdownBuilder.WriteString("\n")
			}
		}

	case "company":
		// 按公司名称搜索
		found := false
		for _, company := range companies {
			if strings.Contains(company.ChineseName, args.Query) ||
				strings.Contains(company.EnglishName, args.Query) {
				found = true
				markdownBuilder.WriteString(fmt.Sprintf("### %s 公司详情\n\n", company.ChineseName))
				markdownBuilder.WriteString(fmt.Sprintf("- 排名: 第%d名\n", company.Rank))
				markdownBuilder.WriteString(fmt.Sprintf("- 英文名: %s\n", company.EnglishName))
				markdownBuilder.WriteString(fmt.Sprintf("- 营收: %s百万美元 (增长率: %s)\n", company.Details.Revenue.Value, company.Details.Revenue.Growth))
				markdownBuilder.WriteString(fmt.Sprintf("- 利润: %s百万美元 (增长率: %s)\n", company.Details.Profit.Value, company.Details.Profit.Growth))
				markdownBuilder.WriteString(fmt.Sprintf("- 资产: %s百万美元\n", company.Details.Assets.Value))
				markdownBuilder.WriteString(fmt.Sprintf("- 股东权益: %s百万美元\n", company.Details.Equity.Value))
				markdownBuilder.WriteString(fmt.Sprintf("- 利润率: %s\n", company.Details.ProfitMargin.Value))
				markdownBuilder.WriteString(fmt.Sprintf("- 资产回报率: %s\n", company.Details.ROA.Value))
				break
			}
		}
		if !found {
			markdownBuilder.WriteString("未找到该公司信息\n")
		}

	case "country":
		// 按国家搜索
		var filteredCompanies []Company
		for _, company := range companies {
			if strings.Contains(company.SearchTag, args.Query) {
				filteredCompanies = append(filteredCompanies, company)
			}
		}
		sort.Slice(filteredCompanies, func(i, j int) bool {
			return filteredCompanies[i].Rank < filteredCompanies[j].Rank
		})

		if len(filteredCompanies) == 0 {
			markdownBuilder.WriteString("未找到该国家的公司\n")
		} else {
			markdownBuilder.WriteString(fmt.Sprintf("### %s公司列表（按排名排序）\n\n", args.Query))
			for _, company := range filteredCompanies {
				markdownBuilder.WriteString(fmt.Sprintf("#### 第%d名: %s\n", company.Rank, company.ChineseName))
				markdownBuilder.WriteString(fmt.Sprintf("- 英文名: %s\n", company.EnglishName))
				markdownBuilder.WriteString(fmt.Sprintf("- 营收: %s百万美元\n", company.Revenue))
				markdownBuilder.WriteString(fmt.Sprintf("- 利润: %s百万美元 (增长率: %s)\n", company.Details.Profit.Value, company.Details.Profit.Growth))
				markdownBuilder.WriteString("\n")
			}
		}

	default:
		return nil, fmt.Errorf("无效的搜索类型")
	}

	return mcp.NewToolResultText(markdownBuilder.String()), nil
}

var CompanySearchTool = hyancieMCP.MustTool(
	"fortune500-search",
	"2024年500强公司搜索",
	companySearchHandler,
)

func AddCompanySearchTools(mcp *server.MCPServer) {
	CompanySearchTool.Register(mcp)
}
