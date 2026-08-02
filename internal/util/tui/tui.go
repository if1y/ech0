// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// 信息样式（每行）
	infoStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.AdaptiveColor{
			Light: "236", Dark: "252",
		})

	// 标题样式
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{
			Light: "#4338ca", Dark: "#FF7F7F",
		})

	// 高亮样式
	highlight = lipgloss.NewStyle().
			Bold(false).
			Italic(true).
			Foreground(lipgloss.AdaptiveColor{
			Light: "#7c3aed", Dark: "#53b7f5ff",
		})

	// 外框
	boxStyle = lipgloss.NewStyle().
			Bold(true).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#fb5151ff")).
			Padding(1, 1).
			Margin(1, 1)
)

const (
	banner = `
    ______     __    ____ 
   / ____/____/ /_  / __ \
  / __/ / ___/ __ \/ / / /
 / /___/ /__/ / / / /_/ / 
/_____/\___/_/ /_/\____/  
`
)

// GetLogoBanner 获取Logo横幅
func GetLogoBanner() string {
	lines := strings.Split(banner, "\n")
	var rendered []string

	colors := []string{
		"#FF7F7F", // 珊瑚红
		"#FFB347", // 桃橙色
		"#FFEB9C", // 金黄色
		"#B8E6B8", // 薄荷绿
		"#87CEEB", // 天空蓝
		"#DDA0DD", // 梅花紫
		"#F0E68C", // 卡其色
	}

	for i, line := range lines {
		color := lipgloss.Color(colors[i%len(colors)])
		style := lipgloss.NewStyle().Foreground(color)
		rendered = append(rendered, style.Render(line))
	}
	gradientBanner := lipgloss.JoinVertical(lipgloss.Left, rendered...)

	full := lipgloss.JoinVertical(lipgloss.Left,
		gradientBanner,
	)

	return full
}

// PrintCLIBanner 打印CLI横幅
func PrintCLIBanner() {
	banner := GetLogoBanner()

	if _, err := fmt.Fprintln(os.Stdout, banner); err != nil {
		fmt.Fprintf(os.Stderr, "failed to print banner: %v\n", err)
	}
}

// PrintCLIInfo 打印CLI信息
func PrintCLIInfo(title, msg string) {
	// 使用 lipgloss 渲染 CLI 信息
	if _, err := fmt.Fprintln(os.Stdout, infoStyle.Render(titleStyle.Render(title)+": "+highlight.Render(msg))); err != nil {
		fmt.Fprintf(os.Stderr, "failed to print cli info: %v\n", err)
	}
}

// CLIInfoItem 是信息框里的一条统计项。Title 为空表示自由文本块（整体缩进，不参与列对齐）。
type CLIInfoItem struct {
	Title string
	Msg   string
}

// CLIBoxHeader 是信息框的标题行：图标 + 主语 + 值（通常是产物路径）。
//
// 标题行与下方统计项分开渲染，因为两者回答的是不同问题——「这是什么、在哪」对「有多少」。
// 图标尤其必须待在这里：emoji 是宽度不定的字形（部分还带变体选择符），一旦混进统计项的
// 标签列，整列对齐就会被它顶歪。
type CLIBoxHeader struct {
	Icon  string
	Title string
	Value string
}

// boxLabelGap 是标签列与值列之间的间距。
const boxLabelGap = 2

// GetCLIPrintWithBox 渲染带边框的信息框：一行标题，下面是标签对齐的统计项。
func GetCLIPrintWithBox(header CLIBoxHeader, items ...CLIInfoItem) string {
	lines := make([]string, 0, len(items)+2)

	if head := renderBoxHeader(header); head != "" {
		lines = append(lines, head)
		if len(items) > 0 {
			lines = append(lines, "")
		}
	}

	// 标签列补齐到最宽一项，让所有值的左端落在同一列，纵向一扫就能比。
	// 用 lipgloss.Width 而非 len：宽字符与转义序列都得算对。
	labelWidth := 0
	for _, item := range items {
		if w := lipgloss.Width(item.Title); w > labelWidth {
			labelWidth = w
		}
	}

	// 有标题时统计项缩进一级，视觉上归属于标题。
	indent := ""
	if header.Title != "" || header.Value != "" {
		indent = "  "
	}

	for _, item := range items {
		lines = append(lines, renderBoxItem(item, labelWidth, indent)...)
	}

	if len(lines) == 0 {
		return ""
	}
	return boxStyle.Render(strings.Join(lines, "\n"))
}

func renderBoxHeader(h CLIBoxHeader) string {
	if strings.TrimSpace(h.Title) == "" && strings.TrimSpace(h.Value) == "" {
		return ""
	}

	var b strings.Builder
	if h.Icon != "" {
		b.WriteString(h.Icon)
		b.WriteString("  ")
	}
	b.WriteString(titleStyle.Render(h.Title))
	if h.Value != "" {
		b.WriteString("  ")
		b.WriteString(highlight.Render(h.Value))
	}
	return infoStyle.Render(b.String())
}

func renderBoxItem(item CLIInfoItem, labelWidth int, indent string) []string {
	parts := strings.Split(item.Msg, "\n")

	if item.Title == "" {
		out := make([]string, 0, len(parts))
		for _, line := range parts {
			out = append(out, infoStyle.Render(indent+highlight.Render(line)))
		}
		return out
	}

	label := indent + titleStyle.Render(item.Title) +
		strings.Repeat(" ", labelWidth-lipgloss.Width(item.Title)+boxLabelGap)
	out := []string{infoStyle.Render(label + highlight.Render(parts[0]))}

	// 续行对齐到值列，多行值不会甩回标签列下面。
	continuation := indent + strings.Repeat(" ", labelWidth+boxLabelGap)
	for _, line := range parts[1:] {
		out = append(out, infoStyle.Render(continuation+highlight.Render(line)))
	}
	return out
}

// PrintCLIWithBox 打印带边框的CLI信息
func PrintCLIWithBox(header CLIBoxHeader, items ...CLIInfoItem) {
	if _, err := fmt.Fprintln(os.Stdout, GetCLIPrintWithBox(header, items...)); err != nil {
		fmt.Fprintf(os.Stderr, "failed to print cli box: %v\n", err)
	}
}

// ClearScreen 清屏函数，根据操作系统执行不同的清屏命令
func ClearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls") // Windows 清屏命令
	} else {
		cmd = exec.Command("clear") // Linux/macOS 清屏命令
	}
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to clear screen: %v\n", err)
	}
}
