# -*- coding: utf-8 -*-
# coding: utf-8
# @Time        : 2025/4/10
# @Author      : liuboyuan
# @Description : MCP 客户端入口文件，包含主程序逻辑

import asyncio

from MCPClient import MCPClient

async def main():
    """
    主函数，负责初始化客户端并启动聊天循环
    """
    # 创建 MCP 客户端实例
    client = MCPClient()
    try:
        # 连接到服务器
        print('connecting to server...')
        await client.connect_to_server()
        
        # 启动聊天循环
        print('chat loop started')
        await client.chat_loop()
    finally:
        # 清理资源
        await client.cleanup()

if __name__ == "__main__":
    # 运行异步主函数
    asyncio.run(main())