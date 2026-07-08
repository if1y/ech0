// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

// ASCII Art Banner
const banner = `
███████╗     ██████╗    ██╗  ██╗     ██████╗
██╔════╝    ██╔════╝    ██║  ██║    ██╔═████╗
█████╗      ██║         ███████║    ██║██╔██║
██╔══╝      ██║         ██╔══██║    ████╔╝██║
███████╗    ╚██████╗    ██║  ██║    ╚██████╔╝
╚══════╝     ╚═════╝    ╚═╝  ╚═╝     ╚═════╝

` as const

// Catppuccin 渐变色（逐行着色）。浏览器控制台用 %c + CSS，比 chalk 的 ANSI 码更贴合 DevTools。
const gradientColors = [
  '#f38ba8', // Catppuccin Pink
  '#fab387', // Catppuccin Peach
  '#f9e2af', // Catppuccin Yellow
  '#a6e3a1', // Catppuccin Green
  '#94e2d5', // Catppuccin Teal
  '#89b4fa', // Catppuccin Blue
  '#cba6f7', // Catppuccin Mauve
  '#f5c2e7', // Catppuccin Flamingo
  '#eba0ac', // Catppuccin Maroon
] as const

function printWelcome(): void {
  const lines = banner.trim().split('\n')
  console.log() // 添加一个空行
  for (const [index, line] of lines.entries()) {
    const color = gradientColors[index % gradientColors.length]
    console.log(`%c${line}`, `color: ${color}`)
  }
  console.log() // 添加一个空行
}

printWelcome()

export { printWelcome }
