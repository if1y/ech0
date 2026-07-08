// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

// Chat 流式问答相关类型（通过命名空间合并扩展 App.Api）。
declare namespace App {
  namespace Api {
    namespace Chat {
      // 检索命中的来源 Echo
      type ChatSource = {
        echo_id: string
        content: string
        username: string
        echo_created: number
        distance: number
        // 命中 Echo 的媒体附件（图片/视频/音频，后端回查填充，整条 File，供前端展示缩略图/类型标志）
        files?: ChatSourceFile[]
        // 命中 Echo 的扩展分享（音乐/网站/位置…），仅用于在来源里展示一个类型标签
        extension?: Ech0.EchoExtension
      }

      // 来源 Echo 的单个文件（对应后端 model/file.File，字段为 FileObject 超集）
      type ChatSourceFile = {
        id: string
        key?: string
        storage_type: File.StorageType
        url: string
        content_type?: string
        category?: File.Category
        size?: number
        width?: number
        height?: number
      }

      // summarize_echos（区间聚合总结）的覆盖度元数据，供「📚 覆盖 N 条 / M 个月」如实展示
      type ChatCoverage = {
        total: number // 区间内命中的 Echo 总数
        returned: number // 实际纳入聚合的条数（受硬上限约束）
        buckets: number // map-reduce 分桶数；1=窗口放得下、整段塞入
        truncated: boolean // 是否因硬上限截断（保留最近）
      }

      // 一条聊天消息（前端会话内）
      type ChatMessage = {
        role: 'user' | 'assistant'
        content: string
        sources?: ChatSource[]
        // Agent 形态下模型本轮发起过的检索关键词（按到达顺序累积，供「正在检索」状态条展示）
        searches?: string[]
        // 区间聚合总结的覆盖度（summarize_echos 命中时填充，供覆盖度状态条展示）
        coverage?: ChatCoverage
        // 仅前端瞬态：本轮流式因传输/服务端 error 中断（区别于「正常 done 但空回复」），
        // 供失败态渲染「重发」入口。
        failed?: boolean
        // 推理模型的思考过程（reasoning）。后端把内联 <think> 或独立 reasoning_content 分流出来，
        // 前端折叠展示「已思考（用时 X 秒）」。随会话持久化（后端字段 reasoning）。
        reasoning?: string
        // 推理耗时（毫秒），后端权威值，随会话持久化（后端字段 reasoning_ms）。
        reasoning_ms?: number
        // 仅前端瞬态：推理是否仍在流式（true→「思考中」；false/缺省→已结束，展示耗时）。不持久化。
        reasoningActive?: boolean
      }

      // SSE 事件载荷
      type StreamEvent =
        | { type: 'searching'; data: { name: string; query: string } }
        | { type: 'sources'; data: ChatSource[] }
        | { type: 'coverage'; data: ChatCoverage }
        | { type: 'reasoning'; data: { text: string } }
        | { type: 'reasoning_done'; data: { duration_ms: number } }
        | { type: 'delta'; data: { text: string } }
        | { type: 'error'; data: { message: string } }
        | { type: 'done'; data: { done: boolean } }
    }
  }
}
