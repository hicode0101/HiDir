package report

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

// sqliteReport 实现SQLite 报告，表结构、语句与错误消息
// 与 dirsearch 的 sqlite_report.py 保持一致（纯 Go 驱动，无 CGO）。
type sqliteReport struct {
	mutex sync.Mutex
}

// openSQLite 打开数据库文件并校验其完整性。
// 与 dirsearch 一致：已存在但不是合法 SQLite 文件时报错；
// 空文件/新文件视为合法。
func openSQLite(file string) (*sql.DB, error) {
	if dir := filepath.Dir(file); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	conn, err := sql.Open("sqlite", file)
	if err != nil {
		return nil, err
	}
	var result string
	if err := conn.QueryRow("PRAGMA integrity_check").Scan(&result); err != nil || result != "ok" {
		conn.Close()
		return nil, fmt.Errorf("%s is not empty or is not a valid SQLite database", file)
	}
	return conn, nil
}

// quoteTable 为表名加引号（与 dirsearch 的 "%s" 写法一致），防止标识符注入。
func quoteTable(table string) (string, error) {
	if strings.Contains(table, "\"") || strings.Contains(table, "\x00") {
		return "", fmt.Errorf("Invalid table name: %s", table)
	}
	return "\"" + table + "\"", nil
}

// initiate 创建输出目录与结果表（表已存在则跳过）。
func (s *sqliteReport) initiate(file, table string) error {
	quoted, err := quoteTable(table)
	if err != nil {
		return err
	}
	conn, err := openSQLite(file)
	if err != nil {
		return fmt.Errorf("Cannot connect to the SQL database: %s", err)
	}
	defer conn.Close()
	_, err = conn.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
            time DATETIME,
            url TEXT,
            status_code INTEGER,
            content_length INTEGER,
            content_type TEXT,
            redirect TEXT
        );`, quoted))
	if err != nil {
		return fmt.Errorf("Cannot connect to the SQL database: %s", err)
	}
	return nil
}

// save 写入一条结果（dirsearch 的 _reuse=False：每次操作独立连接）。
func (s *sqliteReport) save(file, table string, result Result) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	quoted, err := quoteTable(table)
	if err != nil {
		return
	}
	conn, err := openSQLite(file)
	if err != nil {
		return
	}
	defer conn.Close()
	_, _ = conn.Exec(fmt.Sprintf(
		"INSERT INTO %s VALUES (?, ?, ?, ?, ?, ?);", quoted),
		result.Datetime, result.URL, result.Status,
		result.Length, result.Type, result.Redirect)
}
