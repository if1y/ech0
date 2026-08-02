// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lin-snow/ech0/internal/capsule"
	capsuleBuild "github.com/lin-snow/ech0/internal/capsule/build"
	capsuleCheck "github.com/lin-snow/ech0/internal/capsule/check"
	capsuleExport "github.com/lin-snow/ech0/internal/capsule/export"
	capsuleImporter "github.com/lin-snow/ech0/internal/capsule/importer"
	"github.com/lin-snow/ech0/internal/config"
	tuiUtil "github.com/lin-snow/ech0/internal/util/tui"
	versionPkg "github.com/lin-snow/ech0/internal/version"
)

// 默认路径与 spec §9 的命令语法一致。
const (
	DefaultCapsuleDir = "./capsule"
	DefaultDistDir    = "./dist"
	DefaultBaseURL    = "/"
)

// ExportCapsuleOptions 对应 `ech0 export capsule` 的 flag 集合。
type ExportCapsuleOptions struct {
	Output         string
	IncludePrivate bool
	Zip            bool
}

// DoExportCapsule 把当前实例导出为一个胶囊。
func DoExportCapsule(opts ExportCapsuleOptions) error {
	rt, err := newCapsuleRuntime()
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, err := capsuleExport.Run(ctx, capsuleExport.Deps{
		DB:       rt.db,
		Selector: rt.selector(),
		KV:       rt.kv,
	}, capsuleExport.Options{
		Output:         opts.Output,
		IncludePrivate: opts.IncludePrivate,
		Zip:            opts.Zip,
		Generator:      "ech0 v" + versionPkg.Version,
	})
	if err != nil {
		return err
	}

	items := []tuiUtil.CLIInfoItem{
		{Title: "Echoes", Msg: strconv.Itoa(result.Echoes)},
		{Title: "Files", Msg: fmt.Sprintf("%d (external: %d)", result.Files, result.ExternalFiles)},
		{Title: "Comments", Msg: strconv.Itoa(result.Comments)},
		{Title: "Connects", Msg: strconv.Itoa(result.Connects)},
	}
	if result.SkippedPrivate > 0 {
		items = append(items, tuiUtil.CLIInfoItem{
			Title: "Skipped",
			Msg:   strconv.Itoa(result.SkippedPrivate) + " private — use --include-private to carry them",
		})
	}
	tuiUtil.PrintCLIWithBox(
		tuiUtil.CLIBoxHeader{Icon: "📦", Title: "Capsule", Value: result.Path},
		items...,
	)
	return nil
}

// DoCheck 校验一个胶囊并按 spec §7 分级报告。存在错误级问题时返回 error（退出码 1）。
func DoCheck(path string, fix bool) error {
	src, err := capsule.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	_, report, err := capsuleCheck.Run(context.Background(), src, capsuleCheck.Options{Fix: fix})
	if err != nil {
		return err
	}
	printCheckReport(path, report)
	if report.HasErrors() {
		return fmt.Errorf("capsule check failed: %d error(s)", report.Count(capsuleCheck.LevelError))
	}
	return nil
}

// ImportCapsuleOptions 对应 `ech0 import capsule` 的 flag 集合。
type ImportCapsuleOptions struct {
	IncludePrivate bool
	DryRun         bool
}

// DoImportCapsule 把一个胶囊导入当前实例。校验在此前置（spec §7：import 隐式执行同一套校验），
// 有错误级问题就拒绝落库。
func DoImportCapsule(path string, opts ImportCapsuleOptions) error {
	src, err := capsule.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	loaded, report, err := capsuleCheck.Run(context.Background(), src, capsuleCheck.Options{})
	if err != nil {
		return err
	}
	printCheckReport(path, report)
	if report.HasErrors() {
		return fmt.Errorf("refusing to import an invalid capsule: %d error(s)", report.Count(capsuleCheck.LevelError))
	}

	rt, err := newCapsuleRuntime()
	if err != nil {
		return err
	}

	result, err := capsuleImporter.Run(context.Background(), capsuleImporter.Deps{
		DB:       rt.db,
		Tx:       rt.tx,
		Selector: rt.selector(),
		KV:       rt.kv,
	}, loaded, capsuleImporter.Options{
		IncludePrivate: opts.IncludePrivate,
		DryRun:         opts.DryRun,
	})
	if err != nil {
		// 裸抛 "record not found" 会让人以为胶囊坏了。真正要做的是先把站主建出来，
		// 而最常见的诱因是在没有 data/ 的目录里执行——ech0 会就地新建一个空库，
		// 所以这里报绝对路径：走错目录的人一眼就能看出来。
		if errors.Is(err, capsuleImporter.ErrNoOwner) {
			dbPath := config.Config().Database.Path
			if abs, absErr := filepath.Abs(dbPath); absErr == nil {
				dbPath = abs
			}
			return fmt.Errorf(`%w

A capsule carries no accounts, so its content needs an existing owner to belong to.
Database in use: %s

  · Importing into an existing instance? Run this command from that instance's
    directory — ech0 resolves the database relative to the current directory.
  · Setting up a new instance? Create the owner first: run "ech0 serve", open the
    site and complete the setup, then retry`, err, dbPath)
		}
		return err
	}

	header := tuiUtil.CLIBoxHeader{Icon: "📥", Title: "Imported", Value: path}
	if opts.DryRun {
		header = tuiUtil.CLIBoxHeader{Icon: "🔍", Title: "Dry run", Value: path + "  (nothing written)"}
	}
	items := []tuiUtil.CLIInfoItem{
		{Title: "Echoes", Msg: fmt.Sprintf("created %d, skipped %d", result.EchoesCreated, result.EchoesSkipped)},
		{
			Title: "Files",
			Msg: fmt.Sprintf("created %d, reused %d, renamed %d",
				result.FilesCreated, result.FilesReused, result.FilesRenamed),
		},
		{Title: "Tags", Msg: fmt.Sprintf("created %d", result.TagsCreated)},
		{
			Title: "Comments",
			Msg:   fmt.Sprintf("created %d, skipped %d, orphan %d", result.CommentsCreated, result.CommentsSkipped, result.OrphanComments),
		},
	}
	if result.SkippedPrivate > 0 {
		items = append(items, tuiUtil.CLIInfoItem{Title: "Skipped", Msg: strconv.Itoa(result.SkippedPrivate) + " private"})
	}
	if len(result.SiteFieldsFilled) > 0 {
		items = append(items, tuiUtil.CLIInfoItem{Title: "Site filled", Msg: strings.Join(result.SiteFieldsFilled, ", ")})
	}
	if len(result.Renames) > 0 {
		items = append(items, tuiUtil.CLIInfoItem{Title: "Renamed", Msg: strings.Join(result.Renames, "\n")})
	}
	tuiUtil.PrintCLIWithBox(header, items...)

	// 导入不发布事件（spec §11.3）：向量索引不会自动跟进，明说一句免得用户以为坏了。
	if !opts.DryRun && result.EchoesCreated > 0 {
		tuiUtil.PrintCLIInfo("ℹ️  Note", "no events were emitted; rebuild the embedding index from the dashboard to cover imported content")
	}
	return nil
}

// DoBuild 从胶囊编译出可静态部署的只读站点。
func DoBuild(path, output, baseURL string) error {
	src, err := capsule.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	loaded, report, err := capsuleCheck.Run(context.Background(), src, capsuleCheck.Options{})
	if err != nil {
		return err
	}
	printCheckReport(path, report)
	if report.HasErrors() {
		return fmt.Errorf("refusing to build from an invalid capsule: %d error(s)", report.Count(capsuleCheck.LevelError))
	}

	result, err := capsuleBuild.Run(context.Background(), loaded, capsuleBuild.Options{
		Output:  output,
		BaseURL: baseURL,
	})
	if err != nil {
		return err
	}

	tuiUtil.PrintCLIWithBox(
		tuiUtil.CLIBoxHeader{Icon: "🌐", Title: "Static site", Value: result.Path},
		tuiUtil.CLIInfoItem{Title: "Echoes", Msg: strconv.Itoa(result.Echoes)},
		tuiUtil.CLIInfoItem{Title: "Files", Msg: strconv.Itoa(result.Files)},
		tuiUtil.CLIInfoItem{Title: "Comments", Msg: strconv.Itoa(result.Comments)},
	)
	return nil
}

// printCheckReport 把校验结果写到 stderr（stdout 留给产物摘要，便于管道消费）。
func printCheckReport(path string, report *capsuleCheck.Report) {
	for _, fixed := range report.Fixed {
		fmt.Fprintf(os.Stderr, "fixed   %s\n", fixed)
	}
	for _, issue := range report.Issues {
		location := issue.Path
		if issue.Field != "" {
			location += " [" + issue.Field + "]"
		}
		fmt.Fprintf(os.Stderr, "%-7s %s: %s\n", issue.Level.String(), location, issue.Message)
	}

	errs := report.Count(capsuleCheck.LevelError)
	warns := report.Count(capsuleCheck.LevelWarning)
	if errs == 0 && warns == 0 && len(report.Fixed) == 0 {
		tuiUtil.PrintCLIInfo("✅ Capsule OK", path)
		return
	}
	fmt.Fprintf(os.Stderr, "%s: %d error(s), %d warning(s)\n", path, errs, warns)
}
