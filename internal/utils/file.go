package utils

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// File 表示一个文件对象
type File struct {
	path string
}

// NewFile 创建一个新的File对象
func NewFile(path string) *File {
	return &File{path: path}
}

// Exists 检查文件是否存在
func (f *File) Exists() bool {
	_, err := os.Stat(f.path)
	return !os.IsNotExist(err)
}

// IsValid 检查是否为有效文件
func (f *File) IsValid() bool {
	info, err := os.Stat(f.path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// CanRead 检查文件是否可读
func (f *File) CanRead() bool {
	file, err := os.Open(f.path)
	if err != nil {
		return false
	}
	defer file.Close()
	return true
}

// GetLines 读取文件的所有行
func (f *File) GetLines() []string {
	file, err := os.Open(f.path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

// Read 读取文件内容
func (f *File) Read() string {
	content, err := os.ReadFile(f.path)
	if err != nil {
		return ""
	}
	return string(content)
}

// FileUtils 提供文件操作的工具函数
type FileUtils struct{}

// BuildPath 构建路径
func (fu *FileUtils) BuildPath(parts ...string) string {
	return filepath.Join(parts...)
}

// GetAbsPath 获取绝对路径
func (fu *FileUtils) GetAbsPath(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absPath
}

// Parent 获取父目录
func (fu *FileUtils) Parent(path string) string {
	return filepath.Dir(path)
}

// CreateDir 创建目录
func (fu *FileUtils) CreateDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// CanWrite 检查是否可写
func (fu *FileUtils) CanWrite(path string) bool {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return false
	}
	file.Close()
	os.Remove(path)
	return true
}

// GetFilesByExtension 获取指定目录下的所有指定扩展名的文件
func GetFilesByExtension(dirPath, extension string) []string {
	var files []string

	// 遍历目录
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 检查是否为文件且扩展名匹配
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), "."+strings.ToLower(extension)) {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil
	}

	return files
}

// 全局FileUtils实例
var FileUtil = &FileUtils{}
