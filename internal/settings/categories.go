package settings

// WordlistCategories 是词表分类名到 db/categories 下相对路径的映射，
// 与 dirsearch 的 WORDLIST_CATEGORIES 保持一致。
var WordlistCategories = map[string]string{
	"extensions": "extensions.txt",
	"conf":       "conf.txt",
	"vcs":        "vcs.txt",
	"backups":    "backups.txt",
	"db":         "db.txt",
	"logs":       "logs.txt",
	"keys":       "keys.txt",
	"web":        "web.txt",
	"common":     "common.txt",
	"aggressive": "aggressive.txt",

	// PHP
	"php/laravel":     "php/laravel.txt",
	"php/wordpress":   "php/wordpress.txt",
	"php/codeigniter": "php/codeigniter.txt",
	"php/symfony":     "php/symfony.txt",
	"php/yii":         "php/yii.txt",
	"php/cakephp":     "php/cakephp.txt",
	"php/joomla":      "php/joomla.txt",
	"php/drupal":      "php/drupal.txt",
	"php/magento":     "php/magento.txt",

	// .NET
	"dotnet/aspx": "dotnet/aspx.txt",
	"dotnet/mvc":  "dotnet/mvc.txt",
	"dotnet/core": "dotnet/core.txt",

	// ColdFusion
	"coldfusion": "coldfusion/coldfusion.txt",

	// Java
	"java/jsp":    "java/jsp.txt",
	"java/jsf":    "java/jsf.txt",
	"java/spring": "java/spring.txt",

	// Python
	"python/django":  "python/django.txt",
	"python/flask":   "python/flask.txt",
	"python/fastapi": "python/fastapi.txt",

	// Node
	"node/express": "node/express.txt",

	// Infra
	"infra/docker": "infra/docker.txt",
	"infra/k8s":    "infra/k8s.txt",
	"infra/aws":    "infra/aws.txt",
}

// SortedCategoryNames 返回排序后的全部分类名（用于错误提示）。
func SortedCategoryNames() []string {
	names := make([]string, 0, len(WordlistCategories))
	for name := range WordlistCategories {
		names = append(names, name)
	}
	// 简单插入排序，避免引入 sort 依赖的循环
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
	return names
}
