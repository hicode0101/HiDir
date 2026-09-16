package utils

import (
	"os"
	"path/filepath"
)

// AtomicWriteText 通过同目录临时文件原子性地写入文本，
// 对应 dirsearch 的 atomic_write_private_text。
func AtomicWriteText(file, data string) error {
	dir := filepath.Dir(file)
	base := filepath.Base(file)
	tmp, err := os.CreateTemp(dir, "."+base+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.WriteString(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// 保留既有文件权限（新文件默认 0644）
	if info, err := os.Stat(file); err == nil {
		_ = os.Chmod(tmpName, info.Mode().Perm())
	}
	return os.Rename(tmpName, file)
}

// CreateDir 递归创建目录（已存在时忽略）。
func CreateDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}
