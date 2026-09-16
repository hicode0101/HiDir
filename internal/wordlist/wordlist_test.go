package wordlist

import (
	"strings"
	"testing"
	"time"

	"hidir/internal/utils"
)

// testReader 构造内存词表读取器。
func testReader(files map[string]string) LineReader {
	return func(path string) ([]string, error) {
		if data, ok := files[path]; ok {
			lines := strings.Split(data, "\n")
			if len(lines) > 0 && lines[len(lines)-1] == "" {
				lines = lines[:len(lines)-1]
			}
			return lines, nil
		}
		return nil, &readerError{}
	}
}

type readerError struct{}

func (e *readerError) Error() string { return "not found" }

// baseParams 返回基本生成参数。
func baseParams() GenerateParams {
	return GenerateParams{Extensions: []string{"php"}}
}

var fixedTime = time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

// ---- 基础生成 ----

func TestGenerateBasic(t *testing.T) {
	reader := testReader(map[string]string{
		"words.txt": "admin\nlogin\n# comment\n\nhome\n",
	})
	entries, err := Generate([]string{"words.txt"}, baseParams(), reader, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"admin", "login", "home"}
	if len(entries) != len(want) {
		t.Fatalf("entries = %v, want %v", entries, want)
	}
	for i := range want {
		if entries[i] != want[i] {
			t.Errorf("entries[%d] = %q, want %q", i, entries[i], want[i])
		}
	}
}

func TestGenerateLeadingSlashStripped(t *testing.T) {
	reader := testReader(map[string]string{"w.txt": "/admin"})
	entries, _ := Generate([]string{"w.txt"}, baseParams(), reader, nil)
	if len(entries) != 1 || entries[0] != "admin" {
		t.Errorf("entries = %v", entries)
	}
}

func TestGenerateDedup(t *testing.T) {
	reader := testReader(map[string]string{"w.txt": "admin\nadmin\nadmin"})
	entries, _ := Generate([]string{"w.txt"}, baseParams(), reader, nil)
	if len(entries) != 1 {
		t.Errorf("entries = %v", entries)
	}
}

// ---- %EXT% 展开 ----

func TestGenerateExtTag(t *testing.T) {
	// 与 dirsearch 的 dicc.txt 一致：%EXT% 直接替换为扩展名（不带点）
	reader := testReader(map[string]string{"w.txt": "%EXT%\n.%EXT%.bak\nadmin"})
	params := GenerateParams{Extensions: []string{"php", "asp"}}
	entries, _ := Generate([]string{"w.txt"}, params, reader, nil)
	want := []string{"php", "asp", ".php.bak", ".asp.bak", "admin"}
	if len(entries) != len(want) {
		t.Fatalf("entries = %v, want %v", entries, want)
	}
	for i := range want {
		if entries[i] != want[i] {
			t.Errorf("entries[%d] = %q, want %q", i, entries[i], want[i])
		}
	}
}

// ---- force-extensions ----

func TestGenerateForceExtensions(t *testing.T) {
	reader := testReader(map[string]string{"w.txt": "admin\ndir/\nfile.php"})
	params := GenerateParams{Extensions: []string{"php", "asp"}, ForceExtensions: true}
	entries, _ := Generate([]string{"w.txt"}, params, reader, nil)
	// admin：追加 admin/、admin.php、admin.asp；dir/ 不变；file.php 不变
	want := []string{"admin", "admin/", "admin.php", "admin.asp", "dir/", "file.php"}
	if len(entries) != len(want) {
		t.Fatalf("entries = %v, want %v", entries, want)
	}
	for i := range want {
		if entries[i] != want[i] {
			t.Errorf("entries[%d] = %q, want %q", i, entries[i], want[i])
		}
	}
}

// ---- overwrite-extensions ----

func TestGenerateOverwriteExtensions(t *testing.T) {
	reader := testReader(map[string]string{"w.txt": "index.html\nimage.png\nkeep.php"})
	params := GenerateParams{Extensions: []string{"php", "asp"}, OverwriteExtensions: true}
	entries, _ := Generate([]string{"w.txt"}, params, reader, nil)
	found := map[string]bool{}
	for _, e := range entries {
		found[e] = true
	}
	// index.html 的 .html 被覆盖：保留原词条并追加 index.php / index.asp
	if !found["index.php"] || !found["index.asp"] || !found["index.html"] {
		t.Errorf("overwrite entries = %v", entries)
	}
	// image.png 受媒体扩展名保护，不覆盖
	if found["image.php"] {
		t.Errorf("media extension should be protected: %v", entries)
	}
	// keep.php 已是选定扩展名，不覆盖
	if found["keep.asp"] {
		t.Errorf("selected extension should be kept: %v", entries)
	}
}

// ---- 前缀/后缀 ----

func TestGeneratePrefixSuffix(t *testing.T) {
	reader := testReader(map[string]string{"w.txt": "admin\nbackup/\nquery.php?x=1"})
	params := GenerateParams{
		Extensions: []string{"php"},
		Prefixes:   []string{"."},
		Suffixes:   []string{"~"},
	}
	entries, _ := Generate([]string{"w.txt"}, params, reader, nil)
	// 与 dirsearch 一致：前缀与后缀独立追加，不做组合
	want := []string{".admin", "admin~", ".backup/", ".query.php?x=1"}
	if len(entries) != len(want) {
		t.Fatalf("entries = %v, want %v", entries, want)
	}
	for i := range want {
		if entries[i] != want[i] {
			t.Errorf("entries[%d] = %q, want %q", i, entries[i], want[i])
		}
	}
}

// ---- 大小写 ----

func TestCaseTransforms(t *testing.T) {
	reader := testReader(map[string]string{"w.txt": "admin"})

	entries, _ := Generate([]string{"w.txt"}, GenerateParams{Uppercase: true}, reader, nil)
	if entries[0] != "ADMIN" {
		t.Errorf("uppercase = %v", entries)
	}
	entries, _ = Generate([]string{"w.txt"}, GenerateParams{Lowercase: true}, reader, nil)
	if entries[0] != "admin" {
		t.Errorf("lowercase = %v", entries)
	}
	reader2 := testReader(map[string]string{"w.txt": "aDmIn"})
	entries, _ = Generate([]string{"w.txt"}, GenerateParams{Capitalization: true}, reader2, nil)
	if entries[0] != "Admin" {
		t.Errorf("capital = %v", entries)
	}
}

// ---- 排除扩展名 ----

func TestGenerateExcludeExtensions(t *testing.T) {
	reader := testReader(map[string]string{"w.txt": "index.old\nkeep.php\n# old comment"})
	params := GenerateParams{
		Extensions:        []string{"php"},
		ExcludeExtensions: []string{"old"},
	}
	entries, _ := Generate([]string{"w.txt"}, params, reader, nil)
	if len(entries) != 1 || entries[0] != "keep.php" {
		t.Errorf("entries = %v", entries)
	}
}

// ---- 词表上限 ----

func TestGenerateMaxSize(t *testing.T) {
	reader := testReader(map[string]string{"w.txt": "a\nb\nc\nd"})
	params := GenerateParams{MaxSize: 2}
	_, err := Generate([]string{"w.txt"}, params, reader, nil)
	limitErr, ok := err.(*LimitError)
	if !ok {
		t.Fatalf("expected LimitError, got %v", err)
	}
	if limitErr.MaxSize != 2 {
		t.Errorf("limit = %d", limitErr.MaxSize)
	}
	if limitErr.Error() != "Generated wordlist exceeded --wordlist-max-size (2)" {
		t.Errorf("message = %q", limitErr.Error())
	}
}

// ---- 黑名单模式 ----

func TestGenerateBlacklist(t *testing.T) {
	reader := testReader(map[string]string{"w.txt": "admin\ndir/"})
	params := GenerateParams{
		Extensions:      []string{"php"},
		ForceExtensions: true,
		Prefixes:        []string{"."},
		Suffixes:        []string{"~"},
		IsBlacklist:     true,
	}
	entries, _ := Generate([]string{"w.txt"}, params, reader, nil)
	// 黑名单不做扩展名与前后的改写
	if len(entries) != 2 || entries[0] != "admin" || entries[1] != "dir/" {
		t.Errorf("blacklist entries = %v", entries)
	}
}

// ---- 模板占位符 ----

func TestExpandTemplateLine(t *testing.T) {
	// 无占位符
	got := ExpandTemplateLine("admin", []string{"php"}, nil, fixedTime)
	if len(got) != 1 || got[0] != "admin" {
		t.Errorf("plain = %v", got)
	}

	// %EXT% 展开（与 dirsearch 一致：直接替换为扩展名，不带点）
	got = ExpandTemplateLine(".%EXT%", []string{"php", "asp"}, nil, fixedTime)
	if len(got) != 2 || got[0] != ".php" || got[1] != ".asp" {
		t.Errorf("ext = %v", got)
	}

	// 大小写不敏感
	got = ExpandTemplateLine(".%ext%", []string{"php"}, nil, fixedTime)
	if len(got) != 1 || got[0] != ".php" {
		t.Errorf("case-insensitive ext = %v", got)
	}

	// 未知占位符原样保留
	got = ExpandTemplateLine("path%UNKNOWN%", []string{"php"}, nil, fixedTime)
	if len(got) != 1 || got[0] != "path%UNKNOWN%" {
		t.Errorf("unknown = %v", got)
	}

	// 空展开不产出（当有其他有效占位符时）
	got = ExpandTemplateLine("%SUBJECT%/%EXT%", []string{}, nil, fixedTime)
	if len(got) != 0 {
		t.Errorf("empty expansion = %v", got)
	}

	// 日期占位符
	got = ExpandTemplateLine("backup-%YYYY%-%MM%-%DD%", []string{"php"}, nil, fixedTime)
	if len(got) != 1 || got[0] != "backup-2024-06-15" {
		t.Errorf("date = %v", got)
	}

	// 多占位符笛卡尔积
	got = ExpandTemplateLine("%SUBJECT%-%CRUD_OP%", []string{}, nil, fixedTime)
	if len(got) != 15*10 {
		t.Errorf("cartesian = %d, want %d", len(got), 150)
	}

	// API 版本
	got = ExpandTemplateLine("api/%API_VERSION%/users", []string{}, nil, fixedTime)
	if len(got) != 6 {
		t.Errorf("api version = %v", got)
	}

	// SEP 占位符
	got = ExpandTemplateLine("user%SEP%name", []string{}, nil, fixedTime)
	if len(got) != 4 {
		t.Errorf("sep = %v", got)
	}
}

func TestExpandTemplateLineCategory(t *testing.T) {
	loader := func(name string) []string {
		if name == "common" {
			return []string{"admin", "login"}
		}
		return nil
	}
	got := ExpandTemplateLine("%CATEGORY:common%", []string{"php"}, loader, fixedTime)
	if len(got) != 2 || got[0] != "admin" {
		t.Errorf("category = %v", got)
	}
}

func TestIsTemplateToken(t *testing.T) {
	valid := []string{"%EXT%", "ext", "%SUBJECT%", "%DATE%", "%CATEGORY:web%", "%YYYY%"}
	for _, token := range valid {
		if !IsTemplateToken(token) {
			t.Errorf("%q should be a template token", token)
		}
	}
	if IsTemplateToken("%NOT_A_TOKEN%") {
		t.Error("%NOT_A_TOKEN% should be invalid")
	}
}

// ---- 备份路径 ----

func TestGenerateBackupPaths(t *testing.T) {
	paths := GenerateBackupPaths("index.php")
	if len(paths) == 0 {
		t.Fatal("backup paths should be generated")
	}
	found := map[string]bool{}
	for _, p := range paths {
		found[p] = true
	}
	// index.php 是 PHP 文件 → 生成 ~ 与备份扩展名
	if !found["index.php~"] || !found["index.php.bak"] {
		t.Errorf("paths = %v", paths)
	}

	// 媒体文件不生成备份
	if paths := GenerateBackupPaths("logo.png"); paths != nil {
		t.Errorf("media should not generate backups: %v", paths)
	}
	// 无扩展名不生成
	if paths := GenerateBackupPaths("README"); paths != nil {
		t.Errorf("no-extension should not generate backups: %v", paths)
	}
	// 目录不生成
	if paths := GenerateBackupPaths("admin/"); paths != nil {
		t.Errorf("directory should not generate backups: %v", paths)
	}
}

// ---- 分类解析 ----

func TestResolveCategories(t *testing.T) {
	rfs := testResourceFS()

	got, err := ResolveCategories(rfs, "common,web")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("categories = %v", got)
	}

	got, err = ResolveCategories(rfs, "all")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 20 {
		t.Errorf("all categories = %d", len(got))
	}

	got, err = ResolveCategories(rfs, "php/*")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 9 {
		t.Errorf("php/* = %v", got)
	}

	_, err = ResolveCategories(rfs, "nope")
	if err == nil || !strings.Contains(err.Error(), "Unknown wordlist categories: nope") {
		t.Errorf("unknown error = %v", err)
	}

	got, err = ResolveCategories(rfs, "")
	if err != nil || got != nil {
		t.Errorf("empty categories = %v, %v", got, err)
	}
}

// ---- 词表文件解析 ----

func TestResolveFiles(t *testing.T) {
	rfs := testResourceFS()

	// 默认词表
	files, err := ResolveFiles(rfs, ResolveInput{DefaultFile: "dicc.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "dicc.txt" {
		t.Errorf("default files = %v", files)
	}

	// 指定词表 + 分类
	files, err = ResolveFiles(rfs, ResolveInput{
		WordlistsRaw:  "dicc.txt",
		CategoriesRaw: "common",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("files = %v", files)
	}

	// 不存在的词表
	_, err = ResolveFiles(rfs, ResolveInput{WordlistsRaw: "missing.txt"})
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("missing error = %v", err)
	}
}

// ---- 运行时词表 ----

func TestDictionaryClaimAndReset(t *testing.T) {
	dict := NewDictionary([]string{"a", "b", "c"})

	// 顺序领取
	p1, ok := dict.ClaimNext()
	if !ok || p1 != "a" {
		t.Errorf("claim1 = %q, %v", p1, ok)
	}
	p2, _ := dict.ClaimNext()
	if p2 != "b" {
		t.Errorf("claim2 = %q", p2)
	}
	if dict.Index() != 2 {
		t.Errorf("index = %d", dict.Index())
	}

	// 耗尽
	dict.ClaimNext()
	if _, ok := dict.ClaimNext(); ok {
		t.Error("dictionary should be exhausted")
	}

	// 重置
	dict.Reset()
	if dict.Index() != 0 {
		t.Error("reset should restore index")
	}
	p, ok := dict.ClaimNext()
	if !ok || p != "a" {
		t.Errorf("after reset claim = %q, %v", p, ok)
	}
}

func TestDictionaryReleaseAndRequeue(t *testing.T) {
	dict := NewDictionary([]string{"a", "b", "c"})
	dict.ClaimNext() // a
	dict.ClaimNext() // b
	dict.ReleaseClaims([]string{"a", "b"})
	if dict.Index() != 2 {
		t.Errorf("index after release = %d", dict.Index())
	}

	// 暂停场景：未完成的领取放回队列
	dict = NewDictionary([]string{"a", "b", "c"})
	dict.ClaimNext() // a（未完成）
	dict.RequeueClaims()
	p, ok := dict.ClaimNext()
	if !ok || p != "a" {
		t.Errorf("requeued claim = %q, %v", p, ok)
	}
}

func TestDictionaryAddExtra(t *testing.T) {
	dict := NewDictionary([]string{"a"})
	dict.AddExtra("dynamic1", nil)
	dict.AddExtra("dynamic1", nil) // 去重
	dict.AddExtra("#comment", nil) // 非法路径

	count := 0
	for {
		if _, ok := dict.ClaimNext(); !ok {
			break
		}
		count++
	}
	if count != 2 {
		t.Errorf("claims = %d, want 2", count)
	}
}

func TestGetBlacklists(t *testing.T) {
	rfs := testResourceFS()
	params := GenerateParams{IsBlacklist: true}
	bl := GetBlacklists(rfs, params, rfs.GetLines)
	if _, ok := bl[400]; !ok {
		t.Fatal("400 blacklist missing")
	}
	if _, ok := bl[403]; !ok {
		t.Fatal("403 blacklist missing")
	}
	if _, ok := bl[500]; !ok {
		t.Fatal("500 blacklist missing")
	}
	if len(bl[400]) == 0 {
		t.Error("400 blacklist should have entries")
	}
}

// IsValidPath 单测
func TestIsValidPath(t *testing.T) {
	if !IsValidPath("admin.php", nil) {
		t.Error("valid path")
	}
	if IsValidPath("", nil) || IsValidPath("#comment", nil) {
		t.Error("empty/comment invalid")
	}
	if IsValidPath("backup.old", []string{"old"}) {
		t.Error("excluded extension invalid")
	}
	if IsValidPath("backup.old?x=1", []string{"old"}) {
		t.Error("query stripped before extension check")
	}
}

// testResourceFS 返回内嵌资源 FS。
func testResourceFS() *utils.ResourceFS {
	return &utils.ResourceFS{}
}
