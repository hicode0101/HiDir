# HiDir

HiDir 是一个用 Go 语言实现的目录扫描工具，类似于 dirsearch，用于发现网站上的隐藏文件和目录。它能够通过字典暴力破解的方式，快速扫描目标网站，找出可能存在的敏感文件和目录。

## 安装方法

### 从源码构建

1. 确保你已经安装了 Go 1.16 或更高版本
2. 克隆仓库：
   ```bash
git clone https://github.com/yourusername/HiDir.git
cd HiDir
   ```
3. 构建项目：
   ```bash
go build -o hidir ./cmd/hidir
   ```
4. 运行：
   ```bash
./hidir
   ```

## 使用方法

### 基本用法

```bash
./hidir -u https://example.com -e php,html
```

### 命令格式

```bash
Usage: hidir [-u|--url] target [-e|--extensions] extensions [options]
```

## 参数详细说明

### 必选参数

| 参数名 | 缩写 | 数据类型 | 默认值 | 是否必须 | 描述 | 使用实例 |
|--------|------|----------|--------|----------|------|----------|
| url | u | string | - | 否 | 目标 URL，可以使用多个标志 | `-u https://example.com -u https://test.com` |
| url-file | l | string | - | 否 | URL 列表文件 | `-l urls.txt` |
| stdin | - | bool | false | 否 | 从标准输入读取 URL | `cat urls.txt | ./hidir --stdin` |
| cidr | - | string | - | 否 | 目标 CIDR 范围 | `--cidr 192.168.1.0/24` |
| raw | - | string | - | 否 | 从文件加载原始 HTTP 请求 | `--raw request.txt` |
| session | - | string | - | 否 | 会话文件 | `--session session.json` |
| config | - | string | config.ini | 否 | 配置文件路径 | `--config myconfig.ini` |

### 字典设置

| 参数名 | 缩写 | 数据类型 | 默认值 | 是否必须 | 描述 | 使用实例 |
|--------|------|----------|--------|----------|------|----------|
| wordlists | w | string | - | 否 | 自定义字典文件，用逗号分隔 | `-w dict1.txt,dict2.txt` |
| extensions | e | string | - | 否 | 扩展名列表，用逗号分隔 | `-e php,html,asp` |
| force-extensions | f | bool | false | 否 | 为每个字典条目添加扩展名 | `-f` |
| overwrite-extensions | O | bool | false | 否 | 用指定的扩展名覆盖字典中的扩展名 | `-O` |
| exclude-extensions | - | string | - | 否 | 排除的扩展名列表，用逗号分隔 | `--exclude-extensions jsp,asp` |
| remove-extensions | - | bool | false | 否 | 移除所有路径中的扩展名 | `--remove-extensions` |
| prefixes | - | string | - | 否 | 为所有字典条目添加自定义前缀，用逗号分隔 | `--prefixes _,` |
| suffixes | - | string | - | 否 | 为所有字典条目添加自定义后缀，用逗号分隔 | `--suffixes _,` |
| uppercase | U | bool | false | 否 | 将字典条目转换为大写 | `-U` |
| lowercase | L | bool | false | 否 | 将字典条目转换为小写 | `-L` |
| capital | C | bool | false | 否 | 将字典条目首字母大写 | `-C` |

### 通用设置

| 参数名 | 缩写 | 数据类型 | 默认值 | 是否必须 | 描述 | 使用实例 |
|--------|------|----------|--------|----------|------|----------|
| threads | t | int | 0 | 否 | 线程数 | `-t 100` |
| recursive | r | bool | false | 否 | 递归暴力破解 | `-r` |
| deep-recursive | - | bool | false | 否 | 在每个目录深度执行递归扫描 | `--deep-recursive` |
| force-recursive | - | bool | false | 否 | 为每个找到的路径执行递归暴力破解 | `--force-recursive` |
| max-recursion-depth | R | int | 0 | 否 | 最大递归深度 | `-R 3` |
| recursion-status | - | string | - | 否 | 执行递归扫描的有效状态码 | `--recursion-status 200,301` |
| subdirs | - | string | - | 否 | 扫描给定 URL 的子目录 | `--subdirs admin,test` |
| exclude-subdirs | - | string | - | 否 | 在递归扫描期间排除以下子目录 | `--exclude-subdirs temp,backup` |
| include-status | i | string | - | 否 | 包含的状态码 | `-i 200,301,302` |
| exclude-status | x | string | - | 否 | 排除的状态码 | `-x 404,403` |
| exclude-sizes | - | string | - | 否 | 按大小排除响应 | `--exclude-sizes 0,1024` |
| exclude-text | - | []string | - | 否 | 按文本排除响应 | `--exclude-text "Not Found" --exclude-text "Error"` |
| exclude-regex | - | string | - | 否 | 按正则表达式排除响应 | `--exclude-regex "404 Not Found"` |
| exclude-redirect | - | string | - | 否 | 如果正则表达式匹配重定向 URL，则排除响应 | `--exclude-redirect "login.php"` |
| exclude-response | - | string | - | 否 | 排除与该页面响应相似的响应 | `--exclude-response /error.php` |
| skip-on-status | - | string | - | 否 | 当遇到这些状态码时跳过目标 | `--skip-on-status 403,500` |
| min-response-size | - | int | 0 | 否 | 最小响应长度 | `--min-response-size 100` |
| max-response-size | - | int | 0 | 否 | 最大响应长度 | `--max-response-size 10000` |
| max-time | - | int | 0 | 否 | 扫描的最大运行时间 | `--max-time 3600` |
| exit-on-error | - | bool | false | 否 | 发生错误时退出 | `--exit-on-error` |

### 请求设置

| 参数名 | 缩写 | 数据类型 | 默认值 | 是否必须 | 描述 | 使用实例 |
|--------|------|----------|--------|----------|------|----------|
| http-method | m | string | - | 否 | HTTP 方法 | `-m POST` |
| data | d | string | - | 否 | HTTP 请求数据 | `-d "key=value"` |
| data-file | - | string | - | 否 | 包含 HTTP 请求数据的文件 | `--data-file data.txt` |
| header | H | []string | - | 否 | HTTP 请求头 | `-H "X-Forwarded-For: 127.0.0.1" -H "Authorization: Bearer token"` |
| header-file | - | string | - | 否 | 包含 HTTP 请求头的文件 | `--header-file headers.txt` |
| follow-redirects | F | bool | false | 否 | 跟随 HTTP 重定向 | `-F` |
| random-agent | - | bool | false | 否 | 为每个请求选择随机 User-Agent | `--random-agent` |
| auth | - | string | - | 否 | 认证凭证 | `--auth username:password` |
| auth-type | - | string | - | 否 | 认证类型 (basic, bearer) | `--auth-type basic` |
| cert-file | - | string | - | 否 | 包含客户端证书的文件 | `--cert-file cert.pem` |
| key-file | - | string | - | 否 | 包含客户端证书私钥的文件 | `--key-file key.pem` |
| user-agent | - | string | - | 否 | User-Agent | `--user-agent "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"` |
| cookie | - | string | - | 否 | Cookie | `--cookie "session=abc123"` |

### 连接设置

| 参数名 | 缩写 | 数据类型 | 默认值 | 是否必须 | 描述 | 使用实例 |
|--------|------|----------|--------|----------|------|----------|
| timeout | - | float64 | 0 | 否 | 连接超时 | `--timeout 10` |
| delay | - | float64 | 0 | 否 | 请求之间的延迟 | `--delay 0.5` |
| proxy | - | []string | - | 否 | 代理 URL | `--proxy http://proxy.example.com:8080` |
| proxy-file | - | string | - | 否 | 包含代理服务器的文件 | `--proxy-file proxies.txt` |
| proxy-auth | - | string | - | 否 | 代理认证凭证 | `--proxy-auth username:password` |
| replay-proxy | - | string | - | 否 | 用于重放找到的路径的代理 | `--replay-proxy http://proxy.example.com:8080` |
| tor | - | bool | false | 否 | 使用 Tor 网络作为代理 | `--tor` |
| scheme | - | string | - | 否 | 原始请求的方案 | `--scheme https` |
| max-rate | - | int | 0 | 否 | 每秒最大请求数 | `--max-rate 100` |
| retries | - | int | 0 | 否 | 失败请求的重试次数 | `--retries 3` |
| ip | - | string | - | 否 | 服务器 IP 地址 | `--ip 192.168.1.1` |

### 高级设置

| 参数名 | 缩写 | 数据类型 | 默认值 | 是否必须 | 描述 | 使用实例 |
|--------|------|----------|--------|----------|------|----------|
| crawl | - | bool | false | 否 | 在响应中爬取新路径 | `--crawl` |

### 视图设置

| 参数名 | 缩写 | 数据类型 | 默认值 | 是否必须 | 描述 | 使用实例 |
|--------|------|----------|--------|----------|------|----------|
| full-url | - | bool | false | 否 | 输出完整 URL | `--full-url` |
| redirects-history | - | bool | false | 否 | 显示重定向历史 | `--redirects-history` |
| color | - | bool | true | 否 | 彩色输出 | `--color false` |
| quiet-mode | q | bool | false | 否 | 安静模式 | `-q` |

### 输出设置

| 参数名 | 缩写 | 数据类型 | 默认值 | 是否必须 | 描述 | 使用实例 |
|--------|------|----------|--------|----------|------|----------|
| output | o | string | - | 否 | 输出文件 | `-o results.txt` |
| format | - | string | - | 否 | 报告格式 (simple, plain, json, xml, md, csv, html) | `--format json` |
| log | - | string | - | 否 | 日志文件 | `--log hidir.log` |

## 使用示例

### 基本扫描

```bash
./hidir -u https://example.com -e php,html
```

### 递归扫描

```bash
./hidir -u https://example.com -e php -r
```

### 使用自定义字典

```bash
./hidir -u https://example.com -w custom_dict.txt -e php
```

### 多线程扫描

```bash
./hidir -u https://example.com -e php -t 200
```

### 从文件读取 URL

```bash
./hidir -l urls.txt -e php
```

### 保存结果到文件

```bash
./hidir -u https://example.com -e php -o results.txt
```

### 使用代理

```bash
./hidir -u https://example.com -e php --proxy http://proxy.example.com:8080
```

### 使用 Tor 网络

```bash
./hidir -u https://example.com -e php --tor
```

### 自定义 HTTP 头

```bash
./hidir -u https://example.com -e php -H "X-Forwarded-For: 127.0.0.1" -H "Authorization: Bearer token"
```

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。
