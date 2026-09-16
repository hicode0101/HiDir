// HiDir 是一个兼容 dirsearch 功能与命令行参数的目录（路径）暴力扫描工具。
// 本文件是程序入口：解析参数、处理特殊模式（帮助/会话/词表状态）并启动扫描。
package main

import (
	"fmt"
	"os"

	"hidir/internal/controller"
	"hidir/internal/options"
	"hidir/internal/utils"
	"hidir/internal/view"
	"hidir/internal/wordlist"
)

func main() {
	args := os.Args[1:]
	fs := options.OSFilesystem()
	rfs := utils.NewResourceFS()

	opts, err := options.ParseOptions(args, fs, rfs)
	if err != nil {
		handleExitError(err)
	}

	// --list-sessions：列出可恢复的会话并退出
	if opts.ListSessions {
		baseDir := opts.SessionsDir
		if baseDir == "" {
			if home, homeErr := os.UserHomeDir(); homeErr == nil {
				baseDir = home + "/.hidir/sessions"
			} else {
				baseDir = "sessions"
			}
		}
		count := controller.PrintSessionList(baseDir, func(message string) {
			fmt.Println(message)
		})
		if count == 0 {
			os.Exit(0)
		}
		os.Exit(0)
	}

	// --session 与 --session-id 互斥
	if opts.SessionID != "" && opts.SessionFile != "" {
		fmt.Println("Use either --session or --session-id, not both.")
		os.Exit(1)
	}

	// --session-id：按编号解析会话文件
	if opts.SessionID != "" {
		baseDir := opts.SessionsDir
		if baseDir == "" {
			if home, homeErr := os.UserHomeDir(); homeErr == nil {
				baseDir = home + "/.hidir/sessions"
			} else {
				baseDir = "sessions"
			}
		}
		sessionFile, resolveErr := controller.ResolveSessionID(baseDir, opts.SessionID)
		if resolveErr != nil {
			fmt.Println(resolveErr)
			os.Exit(1)
		}
		opts.SessionFile = sessionFile
	}

	// --session：确认恢复会话
	if opts.SessionFile != "" {
		fmt.Println("Loading a session file will override current options.")
		fmt.Print("[c]ontinue / [q]uit: ")
		reader := utils.NewStdinReader()
		if reader.ReadLine() != "c" {
			os.Exit(1)
		}
	}

	// --wordlist-status：显示词表解析状态并退出
	if opts.WordlistStatus {
		showWordlistStatus(opts, rfs)
		os.Exit(0)
	}

	ui := view.NewCLI(colorOf(opts), opts.Quiet, opts.DisableCLI)
	ctrl := controller.NewController(opts, rfs, ui)

	// 恢复会话
	if opts.SessionFile != "" {
		if importErr := ctrl.Import(opts.SessionFile); importErr != nil {
			ui.Error(importErr.Error())
			os.Exit(1)
		}
	}

	if runErr := ctrl.Run(); runErr != nil {
		ui.Error(runErr.Error())
		os.Exit(1)
	}
}

// colorOf 返回是否启用颜色。
func colorOf(opts *options.Options) bool {
	if opts.Color == nil {
		return true
	}
	return *opts.Color
}

// showWordlistStatus 输出词表解析结果。
func showWordlistStatus(opts *options.Options, rfs *utils.ResourceFS) {
	// 校验目标要求与 dirsearch 保持一致：仅词表模式无需 URL
	wordlists, err := wordlist.ResolveFiles(rfs, wordlist.ResolveInput{
		WordlistsRaw:  opts.WordlistsRaw,
		CategoriesRaw: opts.WordlistCategories,
		DefaultFile:   "dicc.txt",
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	opts.Wordlists = wordlists

	lineReader := func(path string) ([]string, error) {
		return rfs.GetLines(path)
	}
	loader := wordlist.NewCategoryLoader(rfs)
	entries, err := wordlist.Generate(wordlists, wordlist.GenerateParams{
		Extensions:          opts.Extensions,
		ForceExtensions:     opts.ForceExtensions,
		OverwriteExtensions: opts.OverwriteExtensions,
		ExcludeExtensions:   opts.ExcludeExtensions,
		Prefixes:            opts.Prefixes,
		Suffixes:            opts.Suffixes,
		Lowercase:           opts.Lowercase,
		Uppercase:           opts.Uppercase,
		Capitalization:      opts.Capitalization,
		MaxSize:             opts.WordlistMaxSize,
	}, lineReader, loader)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Wordlist status")
	fmt.Printf("Files: %d\n", len(wordlists))
	for _, wordlist := range wordlists {
		fmt.Printf("- %s\n", wordlist)
	}
	fmt.Printf("Generated entries: %d\n", len(entries))
	fmt.Printf("Generation limit: %d\n", opts.WordlistMaxSize)
}

// handleExitError 处理解析错误（含帮助/版本输出）。
func handleExitError(err error) {
	exitErr, ok := err.(*options.ExitError)
	if !ok {
		fmt.Println(err)
		os.Exit(1)
	}
	if exitErr.ToStderr {
		fmt.Fprintln(os.Stderr, exitErr.Message)
	} else {
		fmt.Println(exitErr.Message)
	}
	os.Exit(exitErr.Code)
}
