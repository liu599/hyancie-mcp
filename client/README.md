# MCP 图书搜索系统

本项目是一个基于 MCP (Model Context Protocol) 的图书搜索系统，包含客户端和服务器端实现。

## 文件结构

- `MCPClient.py`: MCP 客户端实现，负责与 MCP 服务器通信，处理用户查询请求
- `client.py`: 客户端入口文件，包含主程序逻辑和异步事件循环
- `MCPServer_book_search.py`: MCP 服务器实现，提供图书搜索服务

## 系统架构

1. 客户端 (`MCPClient.py`):
   - 负责与 MCP 服务器建立连接
   - 处理用户输入的查询请求
   - 调用大语言模型处理查询
   - 执行工具调用并处理返回结果

2. 服务器端 (`MCPServer_book_search.py`):
   - 实现图书搜索服务
   - 提供 book_search 工具接口
   - 返回模拟的图书搜索结果

## 环境要求

- Python 3.7+
- 依赖包：
  - openai
  - python-dotenv
  - mcp
  - httpx

## 环境变量配置

1. 复制环境变量示例文件：
   ```bash
   cp .env.example .env
   ```

2. 编辑 `.env` 文件，配置以下环境变量：
   ```
   # OpenAI Compatiable API 配置
   base_url=your_openai_api_base_url
   api_key=your_openai_api_key
   ```

3. 确保环境变量配置正确：
   - `base_url`: OpenAI Compatiable API 的基础 URL
   - `api_key`: OpenAI Compatiable API 的访问密钥

## 模型选择说明

本项目使用了两个不同的模型来处理不同的任务：

1. QwQ-32B 模型：
   - 用于处理用户查询和工具调用
   - 该模型原生支持 tool call 功能
   - 负责解析用户意图并调用适当的工具

2. DeepSeek-R1 模型：
   - 用于对搜索结果进行总结
   - 该模型不原生支持 tool call
   - 负责将搜索结果整理成易读的格式

## 使用说明

1. 确保已安装所有依赖包
2. 按照上述步骤配置环境变量
3. 运行 `python client.py` 启动客户端
4. 输入`搜索我要的书 使用book_search工具`进行图书搜索
5. 尝试不同的提示词, 有一些提示词如果大模型进入不了tool call, 就会中途返回
6. 输入 'quit' 退出程序

```
您的符合条件的图书
《钢铁是怎样炼成的》
《魔法少女小圆》
《线性代数》
《丹特丽安的书架》

```

## 参考文献

- [MCP 中文入门指南](https://github.com/liaokongVFX/MCP-Chinese-Getting-Started-Guide)
  - 本文档参考了 MCP 中文入门指南中的实现方式
  - 使用了 FastMCP 服务器实现
  - 采用了 stdio 传输协议
  - 参考了工具调用的实现方式
