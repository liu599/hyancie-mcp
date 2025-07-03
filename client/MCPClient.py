# -*- coding: utf-8 -*-
# coding: utf-8
# @Time        : 2025/4/10
# @Author      : liuboyuan
# @Description : MCP 客户端实现类，负责与 MCP 服务器通信和工具调用

import json
import os
import sys
from pathlib import Path
from typing import Optional
from contextlib import AsyncExitStack

from openai import OpenAI
from dotenv import load_dotenv

from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

# 加载环境变量
load_dotenv()

# 获取当前脚本所在目录
current_dir = Path(__file__).parent.absolute()
# 获取项目根目录（当前目录的父目录）
project_root = current_dir.parent.absolute()


# 模型配置
MODEL_NAME = "Qwen3-235B-A22B"
base_url = os.getenv('base_url')
api_key = os.getenv('api_key')
exe_path = os.getenv('exe_path')

# 构建可执行文件路径
exe_path = project_root / exe_path

# 打印配置信息
print("="*50)
print(f"base_url: {base_url}")
print(f"api_key: {api_key}")
print("="*50)

class MCPClient:
    """
    MCP 客户端类，用于与 MCP 服务器通信和工具调用
    """
    def __init__(self):
        """
        初始化 MCP 客户端
        """
        self.session: Optional[ClientSession] = None
        self.exit_stack = AsyncExitStack()
        self.client = OpenAI(
            api_key= api_key,
            base_url=base_url,
        )

    async def connect_to_server(self):
        """
        连接到 MCP 服务器
        """
#         server_params = StdioServerParameters(
#             command='python',
#             args=['MCPServer_book_search.py'],
#             env=None
#         )
        
        server_params = StdioServerParameters(
            command=str(exe_path),
            args=[],
            env=None
        )

        # 建立 stdio 传输连接
        stdio_transport = await self.exit_stack.enter_async_context(
            stdio_client(server_params))
        stdio, write = stdio_transport
        
        # 创建客户端会话
        self.session = await self.exit_stack.enter_async_context(
            ClientSession(stdio, write))

        await self.session.initialize()

    async def process_query(self, query: str) -> str:
        """
        处理用户查询请求
        
        Args:
            query: 用户输入的查询字符串
            
        Returns:
            str: 处理结果
        """
        # 系统提示词，用于约束大语言模型的行为
        system_prompt = (
            "You are a helpful assistant."
            "You have the function of online search / jina search / search supported industries / search top ranked companies information by industries/country. "
            "Please MUST call the tool to search the Appropriate information before answering."
            "Please do not lose the user's question information when searching,"
            "and try to maintain the completeness of the question content as much as possible."
            "When there is a date related question in the user's question,"
            "please use the search function directly to search and PROHIBIT inserting specific time."
        )

        messages = [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": query}
        ]

        # 获取所有可用的工具列表
        response = await self.session.list_tools()
        print("=" * 50)
        print('list available tools:')
        print(response)
        print("=" * 50)
        # 生成工具调用描述信息
        available_tools = [{
            "type": "function",
            "function": {
                "name": tool.name,
                "description": tool.description,
                "input_schema": tool.inputSchema
            }
        } for tool in response.tools]
        print("=" * 50)
        print('generated descriptive tools:')
        print(available_tools)
        print("=" * 50)
        
        # 调用大语言模型处理请求
        response = self.client.chat.completions.create(
            model=MODEL_NAME,
            messages=messages,
            tools=available_tools,
            tool_choice="auto",
        )

        # 处理返回内容
        content = response.choices[0]

        print("=" * 50)
        print('LLM Tool Call Response:')
        print(content)
        print("=" * 50)

        if content.finish_reason == "tool_calls":
            # 解析工具调用信息
            tool_call = content.message.tool_calls[0]
            tool_name = tool_call.function.name
            tool_args = json.loads(tool_call.function.arguments)

            # 执行工具调用
            result = await self.session.call_tool(tool_name, tool_args)
            print("=" * 50)
            print(f"\n\n[Calling tool {tool_name} with args {tool_args}]\n\n")
            print(f"tool result: {result}")
            print("=" * 50)

            # 准备工具调用后的处理提示词
            after_tool_message = [
                {
                    "role": "system",
                    "content": f"""
                        现在你要对搜索结果进行总结。
                        根据用户的{query}, 生成符合要求的答案
                    """
                },
                {
                    "role": "user",
                    "content": result.content[0].text,
                }
            ]
            print("=" * 50)

            print("=" * 50)

            # 使用大语言模型处理工具调用结果
            response = self.client.chat.completions.create(
                model="DeepSeek-R1",
                messages=after_tool_message,
            )
            return response.choices[0].message.content
        else:
            # 这里如果没有调用成功就输出错误。 
            return '调用工具失败, 未进入工具tool_calls'

    async def chat_loop(self):
        """
        聊天循环，持续接收用户输入并处理
        """
        while True:
            try:
                query = input("\nQuery: ").strip()

                if query.lower() == 'quit':
                    break

                response = await self.process_query(query)
                print("\n" + response)

            except Exception as e:
                import traceback
                traceback.print_exc()

    async def cleanup(self):
        """
        清理资源
        """
        await self.exit_stack.aclose()