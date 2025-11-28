package core

import (
	"testing"

	"HiDir/internal/parse"
)

func TestControllerSetup(t *testing.T) {
	// 测试用例1：基本设置
	t.Run("BasicSetup", func(t *testing.T) {
		opts := &parse.Options{
			URLs:      []string{"https://example.com"},
			Wordlists: "",
		}

		controller := NewController(opts)
		err := controller.Setup()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	// 测试用例2：带有认证的设置
	t.Run("SetupWithAuth", func(t *testing.T) {
		opts := &parse.Options{
			URLs:      []string{"https://example.com"},
			Wordlists: "",
			Auth:      "username:password",
			AuthType:  "basic",
		}

		controller := NewController(opts)
		err := controller.Setup()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	// 测试用例3：带有代理认证的设置
	t.Run("SetupWithProxyAuth", func(t *testing.T) {
		opts := &parse.Options{
			URLs:       []string{"https://example.com"},
			Wordlists:   "",
			ProxyAuth: "proxyuser:proxypass",
		}

		controller := NewController(opts)
		err := controller.Setup()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

func TestControllerProcessURLs(t *testing.T) {
	// 测试用例1：从URLs处理
	t.Run("ProcessURLsFromURLs", func(t *testing.T) {
		opts := &parse.Options{
			URLs:      []string{"https://example.com", "https://test.com"},
			Wordlists: "",
		}

		controller := NewController(opts)
		// 直接测试processURLs方法
		controller.processURLs()

		if len(controller.targets) != 2 {
			t.Errorf("Expected 2 targets, got %d", len(controller.targets))
		}
	})

	// 测试用例2：从CIDR处理
	t.Run("ProcessURLsFromCIDR", func(t *testing.T) {
		opts := &parse.Options{
			CIDR:      "192.168.1.0/30", // 应该产生2个可用IP（移除网络地址和广播地址后）
			Wordlists: "",
		}

		controller := NewController(opts)
		controller.processURLs()

		if len(controller.targets) != 2 {
			t.Errorf("Expected 2 targets for CIDR, got %d", len(controller.targets))
		}
	})
}

func TestControllerAddDirectory(t *testing.T) {
	// 测试用例1：添加普通目录
	t.Run("AddNormalDirectory", func(t *testing.T) {
		opts := &parse.Options{
			URLs:      []string{"https://example.com"},
			Wordlists: "",
		}

		controller := NewController(opts)
		controller.addDirectory("test")

		if len(controller.directories) != 1 {
			t.Errorf("Expected 1 directory, got %d", len(controller.directories))
		}
		if controller.directories[0] != "test" {
			t.Errorf("Expected directory 'test', got '%s'", controller.directories[0])
		}
	})

	// 测试用例2：添加排除目录
	t.Run("AddExcludedDirectory", func(t *testing.T) {
		opts := &parse.Options{
			URLs:           []string{"https://example.com"},
			Wordlists:       "",
			ExcludeSubdirs: "test",
		}

		controller := NewController(opts)
		controller.addDirectory("test")

		if len(controller.directories) != 0 {
			t.Errorf("Expected 0 directories for excluded directory, got %d", len(controller.directories))
		}
	})

	// 测试用例3：添加重复目录
	t.Run("AddDuplicateDirectory", func(t *testing.T) {
		opts := &parse.Options{
			URLs:      []string{"https://example.com"},
			Wordlists: "",
		}

		controller := NewController(opts)
		controller.addDirectory("test")
		controller.addDirectory("test")

		if len(controller.directories) != 1 {
			t.Errorf("Expected 1 directory for duplicate addition, got %d", len(controller.directories))
		}
	})
}
