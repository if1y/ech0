// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

// 编辑器的状态
export enum Mode {
  ECH0 = 0, // 默认编辑状态
  Panel = 1, // 显示面板状态
  EXTEN = 3, // 处理扩展状态
  Media = 5, // 媒体上传状态（图片/音频/视频）
  TagManage = 6, // 标签管理状态
}

// 扩展类型
export enum ExtensionType {
  MUSIC = 'MUSIC',
  VIDEO = 'VIDEO',
  GITHUBPROJ = 'GITHUBPROJ',
  WEBSITE = 'WEBSITE',
  LOCATION = 'LOCATION',
  TWEET = 'TWEET',
}

// 图片布局
export enum ImageLayout {
  WATERFALL = 'waterfall', // 瀑布流布局
  GRID = 'grid', // 九宫格布局
  HORIZONTAL = 'horizontal', // 横向布局
  CAROUSEL = 'carousel', // 单图轮播布局
  STACK = 'stack', // 堆叠（多行交错、白边、轻微旋转）
}

// 视频布局：目前仅平铺(none)展示，卡片走 media-first、播放器忽略 layout。
// 独立枚举为未来可能的多布局（如剧场/画中画）留结构。
export enum VideoLayout {
  DEFAULT = 'none',
}

// 音频布局：目前仅平铺(none)展示。独立枚举为未来可能的多布局（如紧凑/歌单）留结构。
export enum AudioLayout {
  DEFAULT = 'none',
}

// S3 Service Provider
export enum S3Provider {
  AWS = 'aws',
  ALIYUN = 'aliyun',
  TENCENT = 'tencent',
  MINIO = 'minio',
  R2 = 'r2',
  OTHER = 'other', // 其它默认按 AWS（virtual-hosted）寻址，可用 use_path_style 强制 path-style
}

// OAuth2 Provider
export enum OAuth2Provider {
  GITHUB = 'github',
  GOOGLE = 'google',
  QQ = 'qq',
  CUSTOM = 'custom',
}

// Follow Status
export enum FollowStatus {
  NONE = 'none',
  PENDING = 'pending',
  ACCEPTED = 'accepted',
  REJECTED = 'rejected',
}

// Online Music Service Provider
export enum MusicProvider {
  NETEASE = 'netease', // 网易云音乐
  QQ = 'tencent', // QQ音乐
  APPLE = 'apple', // Apple Music
}

// Access Token Expiration Time
export enum AccessTokenExpiration {
  EIGHT_HOUR_EXPIRY = '8_hours', // 8小时
  ONE_MONTH_EXPIRY = '1_month', // 1个月
  NEVER_EXPIRY = 'never', // 永不过期
}

// Agent LLM 接口协议 —— 仅按协议族区分；OPENAI 同时承担所有 OpenAI 兼容协议（DeepSeek、Qwen、Ollama 等）
export enum AgentProtocol {
  OPENAI = 'openai',
  ANTHROPIC = 'anthropic',
}
