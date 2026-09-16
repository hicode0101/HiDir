package wordlist

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"

	"hidir/internal/settings"
	"hidir/internal/utils"
)

// ResolveInput 汇总词表文件解析所需输入。
type ResolveInput struct {
	// WordlistsRaw 是 -w 原始参数（逗号分隔文件/目录）。
	WordlistsRaw string
	// CategoriesRaw 是 --wordlist-categories 原始参数。
	CategoriesRaw string
	// DefaultFile 是未指定词表时的默认文件（db/dicc.txt）。
	DefaultFile string
}

// ResolveFiles 解析词表文件列表：
// 合并 -w 与分类词表、展开目录、去重并校验文件可读。
// 返回相对数据库目录的文件路径列表。
func ResolveFiles(rfs *utils.ResourceFS, input ResolveInput) ([]string, error) {
	var wordlists []string
	wordlists = append(wordlists, utils.SplitCSV(input.WordlistsRaw)...)

	categories, err := ResolveCategories(rfs, input.CategoriesRaw)
	if err != nil {
		return nil, err
	}
	wordlists = append(wordlists, categories...)

	if len(wordlists) == 0 {
		wordlists = []string{input.DefaultFile}
	}

	// 展开目录
	var expanded []string
	for _, wl := range wordlists {
		if rfs.IsDir(wl) {
			files, err := rfs.WalkFiles(wl)
			if err != nil {
				return nil, err
			}
			expanded = append(expanded, files...)
		} else {
			expanded = append(expanded, wl)
		}
	}

	// 按顺序去重
	var unique []string
	seen := map[string]bool{}
	for _, p := range expanded {
		if seen[p] {
			continue
		}
		seen[p] = true
		unique = append(unique, p)
	}

	// 校验文件存在且可读
	for _, p := range unique {
		if !rfs.Exists(p) {
			return nil, fmt.Errorf("%s does not exist", p)
		}
		if rfs.IsDir(p) {
			return nil, fmt.Errorf("%s is not a file", p)
		}
	}

	return unique, nil
}

// ResolveCategories 解析分类名为词表文件路径列表。
// 支持 "all"/"*"（全部分类）与 "前缀*"（前缀匹配）。
func ResolveCategories(rfs *utils.ResourceFS, categoriesRaw string) ([]string, error) {
	categories := utils.SplitCSV(strings.ToLower(categoriesRaw))
	if len(categories) == 0 {
		return nil, nil
	}

	normalized := make([]string, 0, len(categories))
	includeAll := false
	for _, category := range categories {
		if category == "all" || category == "*" {
			includeAll = true
		}
		normalized = append(normalized, category)
	}

	var resolved []string
	var unknown []string

	if includeAll {
		// 按分类名字典序展开全部分类，保证输出稳定
		names := settings.SortedCategoryNames()
		for _, name := range names {
			resolved = append(resolved, categoryPath(name))
		}
		return resolved, nil
	}

	for _, category := range normalized {
		matched := false
		if strings.HasSuffix(category, "*") {
			prefix := strings.TrimSuffix(category, "*")
			var names []string
			for name := range settings.WordlistCategories {
				if strings.HasPrefix(name, prefix) {
					names = append(names, name)
				}
			}
			sort.Strings(names)
			for _, name := range names {
				resolved = append(resolved, categoryPath(name))
			}
			if len(names) > 0 {
				matched = true
			}
		}
		if matched {
			continue
		}
		if filename, ok := settings.WordlistCategories[category]; ok {
			resolved = append(resolved, path.Join("categories", filename))
			continue
		}
		unknown = append(unknown, category)
	}

	if len(unknown) > 0 {
		return nil, fmt.Errorf("Unknown wordlist categories: %s\nAvailable categories: %s",
			strings.Join(unknown, ", "), strings.Join(settings.SortedCategoryNames(), ", "))
	}

	return resolved, nil
}

// categoryPath 将分类名转换为数据库内路径。
func categoryPath(name string) string {
	filename := settings.WordlistCategories[name]
	return path.Join("categories", filename)
}

// NewCategoryLoader 构造 CATEGORY: 占位符的分类加载器。
func NewCategoryLoader(rfs *utils.ResourceFS) CategoryLoader {
	return func(name string) []string {
		if !validCategoryName(name) {
			return nil
		}
		filename, ok := settings.WordlistCategories[name]
		if !ok {
			filename = name + ".txt"
		}
		categoryPath := path.Join("categories", filename)
		lines, err := rfs.GetLines(categoryPath)
		if err != nil {
			return nil
		}
		var out []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			out = append(out, utils.LstripOnce(line, "/"))
		}
		return out
	}
}

// validCategoryName 校验分类名字符合法（字母数字与 _./-）。
func validCategoryName(name string) bool {
	for _, c := range name {
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		case c == '_' || c == '.' || c == '/' || c == '-':
		default:
			return false
		}
	}
	return true
}

// Dictionary 是运行时词表：支持并发领取（claim）、动态追加与重置。
// 对应 dirsearch 的 lib/core/dictionary.py。
type Dictionary struct {
	mutex      sync.Mutex
	items      []string
	index      int
	extraIndex int
	extra      []string
	claimed    []string

	extraMembership map[string]bool
}

// NewDictionary 基于已生成的词条构建运行时词表。
func NewDictionary(items []string) *Dictionary {
	return &Dictionary{
		items:           items,
		extraMembership: map[string]bool{},
	}
}

// Len 返回初始词条数量。
func (d *Dictionary) Len() int {
	return len(d.items)
}

// Index 返回当前消费位置。
func (d *Dictionary) Index() int {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.index
}

// ClaimNext 领取下一个词条；耗尽时返回 false。
func (d *Dictionary) ClaimNext() (string, bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if len(d.extra) > d.extraIndex {
		p := d.extra[d.extraIndex]
		d.claimed = append(d.claimed, p)
		d.extraIndex++
		return p, true
	}
	if len(d.items) > d.index {
		p := d.items[d.index]
		d.claimed = append(d.claimed, p)
		d.index++
		return p, true
	}
	return "", false
}

// ReleaseClaims 批量释放已完成的领取。
func (d *Dictionary) ReleaseClaims(paths []string) {
	if len(paths) == 0 {
		return
	}
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// 常规情况下按领取顺序完成，直接移除前缀
	if len(d.claimed) >= len(paths) && slicesEqual(d.claimed[:len(paths)], paths) {
		d.claimed = d.claimed[len(paths):]
		return
	}

	// 暂停/取消可能留下乱序子集：保留未匹配的领取以便恢复
	pending := map[string]int{}
	for _, p := range paths {
		pending[p]++
	}
	var remaining []string
	for _, p := range d.claimed {
		if pending[p] > 0 {
			pending[p]--
		} else {
			remaining = append(remaining, p)
		}
	}
	for _, count := range pending {
		if count > 0 {
			panic("wordlist: release of unclaimed path")
		}
	}
	d.claimed = remaining
}

// RequeueClaims 将未完成的领取重新放回队列头部（用于暂停恢复）。
func (d *Dictionary) RequeueClaims() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if len(d.claimed) == 0 {
		return
	}
	rest := append([]string{}, d.extra[d.extraIndex:]...)
	d.extra = append(append(d.extra[:d.extraIndex], d.claimed...), rest...)
	d.claimed = nil
}

// Contains 初始词表中是否包含该词条。
func (d *Dictionary) Contains(item string) bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	for _, item2 := range d.items {
		if item2 == item {
			return true
		}
	}
	return false
}

// AddExtra 加入动态发现的路径（去重，仅在合法时加入）。
func (d *Dictionary) AddExtra(path string, excludeExtensions []string) {
	if !IsValidPath(path, excludeExtensions) {
		return
	}
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.extraMembership == nil {
		d.extraMembership = map[string]bool{}
	}
	if d.extraMembership[path] {
		return
	}
	d.extraMembership[path] = true
	d.extra = append(d.extra, path)
}

// Reset 重置迭代位置并清空动态路径。
func (d *Dictionary) Reset() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.index = 0
	d.extraIndex = 0
	d.extra = nil
	d.extraMembership = map[string]bool{}
	d.claimed = nil
}

// slicesEqual 比较两个字符串切片是否相等。
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Blacklists 是状态码到黑名单词表的映射。
type Blacklists map[int][]string

// GetBlacklists 从数据库加载 400/403/500 黑名单词表。
func GetBlacklists(rfs *utils.ResourceFS, params GenerateParams, lineReader LineReader) Blacklists {
	out := Blacklists{}
	for _, status := range []int{400, 403, 500} {
		file := fmt.Sprintf("%d_blacklist.txt", status)
		if !rfs.Exists(file) {
			continue
		}
		blacklistParams := params
		blacklistParams.IsBlacklist = true
		items, err := Generate([]string{file}, blacklistParams, lineReader, nil)
		if err != nil {
			continue
		}
		out[status] = items
	}
	return out
}
