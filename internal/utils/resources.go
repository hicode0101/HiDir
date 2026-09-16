package utils

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"hidir/assets"
)

// ResourceFS 抽象词表数据库的读取来源：优先使用磁盘上的 db 目录，
// 否则回退到二进制内嵌资源。
type ResourceFS struct {
	// Dir 是磁盘上生效的数据库目录；为空表示使用内嵌资源。
	Dir string
}

// dbCandidates 返回可能的磁盘数据库目录（按优先级）。
func dbCandidates() []string {
	var candidates []string
	if env := os.Getenv("HIDIR_DB"); env != "" {
		candidates = append(candidates, env)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "db"))
	}
	candidates = append(candidates, "db")
	return candidates
}

// NewResourceFS 定位词表数据库目录。
func NewResourceFS() *ResourceFS {
	for _, dir := range dbCandidates() {
		if info, err := os.Stat(filepath.Join(dir, "dicc.txt")); err == nil && !info.IsDir() {
			return &ResourceFS{Dir: dir}
		}
	}
	return &ResourceFS{}
}

// joined 将路径转换为磁盘绝对路径或内嵌 FS 路径。
// 绝对路径直接走磁盘（用户指定的词表/文件）；
// 相对路径按数据库目录解析（内建词表资源）。
func (r *ResourceFS) joined(rel string) (diskPath string, fsPath string) {
	rel = filepath.ToSlash(rel)
	if filepath.IsAbs(rel) || strings.Contains(rel, ":") {
		return rel, ""
	}
	if r.Dir != "" {
		return filepath.Join(r.Dir, filepath.FromSlash(rel)), ""
	}
	return "", assets.DBDir + "/" + strings.TrimPrefix(rel, "/")
}

// Exists 判断资源是否存在。
func (r *ResourceFS) Exists(rel string) bool {
	disk, embed := r.joined(rel)
	if disk != "" {
		_, err := os.Stat(disk)
		return err == nil
	}
	_, err := fs.Stat(assets.FS, embed)
	return err == nil
}

// IsDir 判断资源是否为目录。
func (r *ResourceFS) IsDir(rel string) bool {
	disk, embed := r.joined(rel)
	if disk != "" {
		info, err := os.Stat(disk)
		return err == nil && info.IsDir()
	}
	info, err := fs.Stat(assets.FS, embed)
	return err == nil && info.IsDir()
}

// ReadFile 读取资源内容。
func (r *ResourceFS) ReadFile(rel string) ([]byte, error) {
	disk, embed := r.joined(rel)
	if disk != "" {
		return os.ReadFile(disk)
	}
	return fs.ReadFile(assets.FS, embed)
}

// GetLines 按行读取资源文件（与 Python splitlines 一致，末尾空行不计）。
func (r *ResourceFS) GetLines(rel string) ([]string, error) {
	data, err := r.ReadFile(rel)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, nil
}

// ListDir 列出目录下的所有条目（不递归）。
func (r *ResourceFS) ListDir(rel string) ([]string, error) {
	disk, embed := r.joined(rel)
	if disk != "" {
		entries, err := os.ReadDir(disk)
		if err != nil {
			return nil, err
		}
		var out []string
		for _, e := range entries {
			out = append(out, filepath.ToSlash(filepath.Join(rel, e.Name())))
		}
		return out, nil
	}
	entries, err := fs.ReadDir(assets.FS, embed)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		out = append(out, rel+"/"+e.Name())
	}
	return out, nil
}

// WalkFiles 递归列出目录下的全部文件。
func (r *ResourceFS) WalkFiles(rel string) ([]string, error) {
	disk, embed := r.joined(rel)
	var out []string
	if disk != "" {
		root := disk
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				relPath, relErr := filepath.Rel(root, path)
				if relErr != nil {
					return relErr
				}
				out = append(out, filepath.Join(rel, filepath.FromSlash(filepath.ToSlash(relPath))))
			}
			return nil
		})
		return out, err
	}
	err := fs.WalkDir(assets.FS, embed, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			out = append(out, strings.TrimPrefix(path, assets.DBDir+"/"))
		}
		return nil
	})
	return out, err
}

// FormatDatetimeForPath 将时间字符串转换为可作路径的安全格式。
func FormatDatetimeForPath(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, " ", "_"), ":", "-")
}
