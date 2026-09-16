# HiDir

HiDir 是一个使用 Go 语言编写的高性能 Web 路径（目录/文件）暴力扫描工具，兼容 [dirsearch](https://github.com/maurosoria/dirsearch) 的功能与全部命令行参数。它内置了 dirsearch 的完整词表数据库（`db/dicc.txt`、分类词表、状态码黑名单、User-Agent 列表），并作为资源嵌入二进制文件中，编译后即可独立运行，无需安装 Python 环境与第三方依赖。

## 功能特性

- **多种目标输入方式**：单个/多个 URL（`-u`）、URL 列表文件（`-l`）、标准输入（`--stdin`）、CIDR 网段（`--cidr`）、原始 HTTP 请求文件（`--raw`）、Nmap XML 报告（`--nmap-report`）
- **强大的词表引擎**：
  - `%EXT%` 等模板占位符展开（`%EXT%`、`%SUBJECT%`、`%CRUD_OP%`、`%AUTH_OP%`、`%ADMIN_OP%`、`%ENV%`、`%DB%`、`%ARCHIVE%`、`%BACKUP%`、`%API_VERSION%`、`%YYYY%`/`%MM%`/`%DD%`/`%DATE%`、`%CATEGORY:名称%` 等）
  - 强制扩展名（`-f`）、覆盖扩展名（`--overwrite-extensions`）、排除扩展名
  - 自定义前缀/后缀、大写/小写/首字母大写变换
  - 多词表合并、目录递归展开、按序去重、`--wordlist-max-size` 上限保护
  - 30 余个内置词表分类（common、web、php/*、java/*、python/*、infra/* 等）
- **智能过滤（减少误报）**：
  - 通配符（软 404）自动检测：随机路径采样 + 动态内容 diff + 重定向正则匹配
  - 自动校准（`--auto-calibration`）与重复响应指纹过滤
  - 状态码黑名单词表（400/403/500）
  - include/exclude 状态码（支持范围）、按大小/文本/正则/跳转地址排除
  - 高级匹配器/过滤器（对齐 ffuf/wfuzz）：`--match-status`、`--filter-size`、`--match-words`、`--match-lines`、`--match-regex`、`--match-header`、`--match-time` 等，支持 and/or 组合模式
  - `--filter-threshold` 重复响应阈值
- **递归与爬取**：递归扫描（`-r`）、逐层递归（`--deep-recursive`）、强制递归（`--force-recursive`）、递归深度与状态码控制、子目录扫描（`--subdirs`）、响应爬取（`--crawl`，支持 HTML/text/robots.txt）、备份文件探测（`--find-backup`）
- **完整的请求能力**：GET/POST 等全部方法、请求体（含 `--data-file`）、自定义头（含 `--headers-file`）、Cookie、随机 User-Agent、Basic/Digest/Bearer/JWT 认证、客户端证书、跟随重定向
- **连接控制**：超时、请求延迟、并发线程、HTTP/SOCKS5 代理、Tor、重放代理（`--replay-proxy`）、每秒最大请求数（`--max-rate`）、失败重试、`--ip` 指定服务器 IP
- **多种报告格式**：simple、plain、json、xml、md（Markdown）、csv、html（交互式单页报告），支持 `{format}`/`{extension}`/`{date}`/`{host}`/`{scheme}`/`{port}` 路径变量；响应体保存（`--save-response`/`--save-response-jsonl`）
- **会话支持**：Ctrl+C 暂停交互菜单（继续/下一个目录/跳过目标/保存退出）、会话保存与恢复（`-s`/`--session-id`/`--list-sessions`）
- **彩色终端输出**：按状态码着色的结果报告、实时进度条、quiet/disable-cli/verbose 输出模式

## 安装与构建

需要 Go 1.25 及以上版本：

```bash
cd HiDir
go build -o hidir .
```

生成单个可执行文件 `hidir`（Windows 下为 `hidir.exe`），词表数据库已嵌入，可直接拷贝使用。若工作目录或可执行文件同目录存在 `db/` 目录，则优先使用磁盘上的词表，便于自定义；也可通过环境变量 `HIDIR_DB` 指定词表目录。

## 快速开始

```bash
# 最简单的扫描
hidir -u https://example.com/

# 指定扩展名与线程数
hidir -u https://example.com/ -e php,html,js -t 50

# 递归扫描，强制扫描所有发现的路径
hidir -u https://example.com/ -e php -r -f

# 从文件读取多个目标并输出 JSON 报告
hidir -l urls.txt -e php -O json -o result.json

# 带认证与代理扫描
hidir -u https://example.com/ -e php --auth "admin:password" --auth-type basic -p socks5://127.0.0.1:9050
```

## 全部参数说明

### Mandatory（必选参数）

| 参数 | 说明 |
|------|------|
| `-u URL`, `--url=URL` | 目标 URL，可多次使用 |
| `-l PATH`, `--urls-file=PATH` | URL 列表文件 |
| `--stdin` | 从标准输入读取 URL |
| `--cidr=CIDR` | CIDR 网段作为目标 |
| `--raw=PATH` | 从文件加载原始 HTTP 请求（配合 `--scheme` 指定协议） |
| `--nmap-report=PATH` | 从 nmap XML 报告提取目标（建议 nmap 扫描时带 `-sV`） |
| `-s`, `--session=SESSION_FILE` | 恢复会话文件 |
| `--session-id=ID` | 按编号恢复会话（配合 `--list-sessions`） |
| `--config=PATH` | 配置文件路径（默认 `DIRSEARCH_CONFIG` 环境变量，否则 `config.ini`） |

### Dictionary Settings（字典设置）

| 参数 | 说明 |
|------|------|
| `-w`, `--wordlists=WORDLISTS` | 词表文件或目录（逗号分隔） |
| `--wordlist-categories=CATEGORIES` | 内置分类名（如 `common,conf,web`；`all` 为全部；支持 `php/*` 前缀通配） |
| `--wordlist-backend=BACKEND` | 词表生成后端：`auto`、`python`、`native`（默认 `auto`） |
| `--wordlist-status` | 显示词表解析结果与条目数后退出 |
| `--wordlist-max-size=COUNT` | 生成词表的最大条目数（默认 500000） |
| `-e`, `--extensions=EXTENSIONS` | 扩展名列表（逗号分隔）；`'*'` 表示常见扩展名集合 |
| `-f`, `--force-extensions` | 为每个词表条目追加扩展名（默认仅替换 `%EXT%` 关键字） |
| `--overwrite-extensions` | 用选定扩展名覆盖词表中的未知扩展名 |
| `--exclude-extensions=EXTENSIONS` | 排除的扩展名列表 |
| `--prefixes=PREFIXES` | 为所有词条添加前缀（逗号分隔） |
| `--suffixes=SUFFIXES` | 为所有词条添加后缀（忽略目录，逗号分隔） |
| `-U`, `--uppercase` | 词表全部大写 |
| `-L`, `--lowercase` | 词表全部小写 |
| `-C`, `--capital` | 词表首字母大写 |

### General Settings（通用设置）

| 参数 | 说明 |
|------|------|
| `-t`, `--threads=THREADS` | 并发线程数（默认 25） |
| `--list-sessions` | 列出可恢复的会话后退出 |
| `--sessions-dir=PATH` | 会话存储目录 |
| `-a`, `--async` | 启用异步模式 |
| `--sync`, `--no-async` | 使用同步模式 |
| `-r`, `--recursive` | 递归扫描 |
| `--deep-recursive` | 对每一层目录深度递归（如 `api/users` → `api/`） |
| `--force-recursive` | 对发现的每个路径都递归（不仅是目录） |
| `-R`, `--max-recursion-depth=DEPTH` | 最大递归深度（0 为无限） |
| `--recursion-status=CODES` | 触发递归的状态码范围（默认 `200-399,401,403`） |
| `--filter-threshold=THRESHOLD` | 重复响应被过滤前的最大次数 |
| `--subdirs=SUBDIRS` | 扫描 URL 的子目录（逗号分隔，如 `/，api/`） |
| `--exclude-subdirs=SUBDIRS` | 递归时排除的子目录 |
| `-i`, `--include-status=CODES` | 仅包含的状态码（如 `200,300-399`） |
| `-x`, `--exclude-status=CODES` | 排除的状态码（如 `301,500-599`） |
| `--exclude-sizes=SIZES` | 按响应大小排除（如 `0,0B,4KB`） |
| `--exclude-text=TEXTS` | 按响应文本排除（可多次使用） |
| `--exclude-regex=REGEX` | 按正则排除响应 |
| `--exclude-redirect=STRING` | 跳转地址匹配该正则/文本时排除 |
| `--exclude-response=PATH` | 排除与指定页面相似的响应 |
| `--skip-on-status=CODES` | 命中这些状态码时跳过整个目标（默认 `429`） |
| `--min-response-size=LENGTH` | 最小响应长度（如 `1024`、`1KB`） |
| `--max-response-size=LENGTH` | 最大响应长度 |
| `--max-time=SECONDS` | 整个扫描的最长运行时间 |
| `--target-max-time=SECONDS` | 单个目标的最长运行时间 |
| `--exit-on-error` | 出错时立即退出 |

### Advanced Filtering（高级过滤）

| 参数 | 说明 |
|------|------|
| `--auto-calibration` | 强制从开始就进行额外通配符校准 |
| `--matcher-mode=MODE`, `--mmode` | 匹配器组合方式：`and`、`or`（默认 `or`） |
| `--filter-mode=MODE`, `--fmode` | 过滤器组合方式：`and`、`or`（默认 `or`） |
| `--match-status=CODES`, `--mc` | 状态码匹配器 |
| `--filter-status=CODES`, `--fc` | 状态码过滤器 |
| `--match-size=SIZES`, `--ms` | 响应长度匹配器 |
| `--filter-size=SIZES`, `--fs` | 响应长度过滤器 |
| `--match-words=WORDS`, `--mw` | 响应词数匹配器 |
| `--filter-words=WORDS`, `--fw` | 响应词数过滤器 |
| `--match-lines=LINES`, `--ml` | 响应行数匹配器 |
| `--filter-lines=LINES`, `--fl` | 响应行数过滤器 |
| `--match-regex=REGEX`, `--mr` | 响应体正则匹配器 |
| `--filter-regex=REGEX`, `--fr` | 响应体正则过滤器 |
| `--match-header=TEXT` | 响应头文本匹配器（可多次使用） |
| `--filter-header=TEXT` | 响应头文本过滤器（可多次使用） |
| `--match-header-regex=REGEX` | 响应头正则匹配器 |
| `--filter-header-regex=REGEX` | 响应头正则过滤器 |
| `--match-time=TIME`, `--mt` | 响应耗时匹配器（毫秒，如 `>100`、`<100`） |
| `--filter-time=TIME`, `--ft` | 响应耗时过滤器 |

### Request Settings（请求设置）

| 参数 | 说明 |
|------|------|
| `-m`, `--http-method=METHOD` | HTTP 方法（默认 GET） |
| `--request-backend=BACKEND` | 请求后端：`python`、`native`（默认 `python`；Go 版本两者等价） |
| `-d`, `--data=DATA` | HTTP 请求体 |
| `--data-file=PATH` | 从文件读取请求体（不转义、不转换换行） |
| `-H`, `--header=HEADER` | 自定义请求头（可多次使用） |
| `--headers-file=PATH` | 从文件读取请求头 |
| `-F`, `--follow-redirects` | 跟随 HTTP 重定向 |
| `--random-agent` | 每个请求随机选择 User-Agent |
| `--auth=CREDENTIAL` | 认证凭据（如 `user:password` 或 Bearer token） |
| `--auth-type=TYPE` | 认证类型：`basic`、`digest`、`bearer`、`ntlm`、`jwt` |
| `--cert-file=PATH` | 客户端证书文件 |
| `--key-file=PATH` | 客户端证书私钥文件（未加密） |
| `--user-agent=UA` | User-Agent |
| `--cookie=COOKIE` | Cookie |

### Connection Settings（连接设置）

| 参数 | 说明 |
|------|------|
| `--timeout=SECONDS` | 连接超时秒数（大于 0，默认 7.5） |
| `--delay=SECONDS` | 请求间隔秒数（大于等于 0） |
| `-p`, `--proxy=PROXY` | 代理 URL（HTTP/SOCKS5，可多次使用） |
| `--proxies-file=PATH` | 代理列表文件 |
| `--proxy-auth=CREDENTIAL` | 代理认证凭据 |
| `--replay-proxy=PROXY` | 使用该代理重放发现的路径 |
| `--tor` | 使用 Tor 网络作为代理（SOCKS5 127.0.0.1:9050/9150） |
| `--scheme=SCHEME` | 指定协议（默认自动探测） |
| `--max-rate=RATE` | 每秒最大请求数（0 为不限） |
| `--retries=RETRIES` | 失败重试次数（默认 1） |
| `--ip=IP` | 直接指定服务器 IP（绕过 DNS） |
| `--interface=IFACE` | 绑定网络接口（仅 Linux 支持） |

### Advanced Settings（高级设置）

| 参数 | 说明 |
|------|------|
| `--crawl` | 从响应中爬取新路径 |
| `--find-backup` | 探测已发现文件的备份（`.bak`、`~`、`.zip` 等） |

### View Settings（视图设置）

| 参数 | 说明 |
|------|------|
| `--full-url` | 输出完整 URL（quiet 模式自动启用） |
| `--redirects-history` | 显示重定向历史 |
| `--no-color` | 关闭彩色输出 |
| `-q`, `--quiet-mode` | 安静模式（仅显示结果） |
| `--disable-cli` | 关闭命令行输出 |
| `-v`, `--verbose` | 显示响应耗时与 Content-Type |

### Output Settings（输出设置）

| 参数 | 说明 |
|------|------|
| `-O`, `--output-formats=FORMAT` | 报告格式（逗号分隔）：`simple`、`plain`、`json`、`xml`、`md`、`csv`、`html`、`sqlite` |
| `-o`, `--output-file=PATH` | 输出文件路径，支持 `{format}`、`{extension}`、`{date}`、`{datetime}`、`{host}`、`{scheme}`、`{port}` 变量 |
| `--mysql-url=URL` | MySQL 输出（见下方"与 dirsearch 的差异"） |
| `--postgres-url=URL` | PostgreSQL 输出（同上） |
| `--save-response=PATH` | 将命中的响应体保存到目录 |
| `--save-response-jsonl=PATH` | 将命中的响应追加到 JSONL 文件 |
| `--log=PATH` | 日志文件 |

### 帮助

| 参数 | 说明 |
|------|------|
| `-h`, `--help` | 显示常用选项帮助 |
| `-hh`, `--help-all` | 显示全部选项帮助 |
| `--version` | 显示版本号 |

## 使用示例

### 基础扫描

```bash
# 扫描目标，使用默认词表（dicc.txt），无扩展名
hidir -u https://example.com/

# 常见扩展名（等价于 php,jsp,asp,aspx,do,action,cgi,html,htm,js,tar.gz）
hidir -u https://example.com/ -e '*'

# 指定扩展名 + 50 线程 + 递归
hidir -u https://example.com/ -e php,asp -t 50 -r

# %EXT% 词表：dicc.txt 中包含 %EXT% 的条目会为每个扩展名生成一行
hidir -u https://example.com/ -e php -w db/dicc.txt
```

### 强制扩展名与前后缀

```bash
# 为所有词条追加 .php（admin → admin.php、admin/）
hidir -u https://example.com/ -e php -f

# 添加前缀 . 与后缀 ~（备份文件探测思路）
hidir -u https://example.com/ -e php --prefixes '.' --suffixes '~'

# 大小写变换
hidir -u https://example.com/ -w words.txt -U   # 全大写
hidir -u https://example.com/ -w words.txt -C   # 首字母大写
```

### 词表分类

```bash
# 使用内置分类词表
hidir -u https://example.com/ --wordlist-categories common,web

# 全部 PHP 相关分类
hidir -u https://example.com/ --wordlist-categories 'php/*'

# 查看词表解析结果（不发起扫描）
hidir --wordlist-status --wordlist-categories common -e php
```

### 多目标与子目录

```bash
# 多个目标
hidir -u https://example.com/ -u https://example.org/ -e php

# 从文件读取目标（支持 # 注释）
hidir -l urls.txt -e php

# 扫描目标下的 /api/ 与 /admin/ 子目录
hidir -u https://example.com/ --subdirs api/,admin/ -e php

# 整个 C 段
hidir --cidr 192.168.1.0/24 -e php -t 100
```

### 过滤与高级匹配

```bash
# 只显示 200 与 3xx，排除 0 字节与 4KB 响应
hidir -u https://example.com/ -e php -i 200-399 --exclude-sizes 0,4KB

# 排除含 "not found" 文本的响应
hidir -u https://example.com/ -e php --exclude-text "not found" --exclude-regex "^404$"

# ffuf 风格：只保留状态 200 且长度在 100-5000 之间的响应
hidir -u https://example.com/ -e php --match-status 200 --match-size 100-5000 --mmode and

# 过滤掉耗时超过 2 秒的响应
hidir -u https://example.com/ -e php --filter-time '>2000'
```

### 递归与爬取

```bash
# 递归扫描（默认递归状态码 200-399,401,403），最多 3 层
hidir -u https://example.com/ -e php -r -R 3

# 每层目录都递归 + 对文件路径也递归
hidir -u https://example.com/ -e php -r --deep-recursive --force-recursive

# 爬取响应中的链接并加入扫描队列
hidir -u https://example.com/ -e php --crawl

# 探测备份文件（admin.php → admin.php.bak、admin.php~、admin.zip ...）
hidir -u https://example.com/ -e php --find-backup
```

### 认证与请求定制

```bash
# Basic 认证
hidir -u https://example.com/ -e php --auth "user:password" --auth-type basic

# Bearer Token
hidir -u https://example.com/ -e php --auth "eyJhbGciOi..." --auth-type bearer

# POST 请求 + 自定义头 + Cookie
hidir -u https://example.com/ -m POST -d "username=admin&password=admin" \
  -H "X-Forwarded-For: 127.0.0.1" --cookie "session=abc123"

# 从原始请求文件扫描（配合 Burp 抓包）
hidir --raw request.txt --scheme https
```

### 代理与限速

```bash
# 通过 SOCKS5 代理扫描
hidir -u https://example.com/ -e php -p socks5://127.0.0.1:1080

# Tor 网络
hidir -u https://example.com/ -e php --tor

# 限速：每秒最多 20 个请求，每个请求间隔 0.1 秒
hidir -u https://example.com/ -e php --max-rate 20 --delay 0.1

# 将发现的路径通过 Burp 重放
hidir -u https://example.com/ -e php --replay-proxy http://127.0.0.1:8080

# 直接指定服务器 IP（绕过 DNS / 与 Host 头搭配）
hidir -u https://example.com/ -e php --ip 93.184.216.34
```

### 报告输出

```bash
# 同时输出多种格式（用 {format} 变量区分文件）
hidir -u https://example.com/ -e php -O simple,json,html -o 'reports/example.{format}'

# 按主机与日期命名报告
hidir -u https://example.com/ -e php -O json -o 'out/{host}-{date}.{extension}'

# 保存命中的响应体
hidir -u https://example.com/ -e php --save-response responses/

# 安静模式 + 完整 URL（便于管道处理）
hidir -u https://example.com/ -e php -q --full-url | grep 200
```

### 会话管理

```bash
# 列出可恢复的会话
hidir --list-sessions

# 恢复指定会话文件
hidir -s sessions/2024-06-15/session_2024-06-15_12-00-00.json

# 扫描过程中按 Ctrl+C 暂停，可选择：
#   [c]ontinue  继续扫描
#   [n]ext      跳到下一个目录任务
#   [s]kip      跳过当前目标
#   [q]uit      退出（可再选 [s]ave 保存会话 / [q]uit 直接退出）
```

### 组合示例

```bash
# 典型的完整扫描流程
hidir -u https://example.com/ \
      -w db/dicc.txt \
      -e php,asp,aspx,jsp,html,js \
      -t 50 \
      -r -R 2 \
      --exclude-sizes 0 \
      --exclude-texts "404 Not Found" \
      -O json,plain \
      -o 'reports/{host}-{date}.{format}' \
      --random-agent \
      --max-rate 100 \
      -v
```

## 配置文件

HiDir 支持与 dirsearch 相同格式的 `config.ini`。默认按以下顺序查找：
`--config` 参数 → `HIDIR_CONFIG`/`DIRSEARCH_CONFIG` 环境变量 → 当前目录 `config.ini` → 可执行文件同目录 `config.ini` → 内置默认配置。

```ini
[general]
threads = 25
recursive = False
recursion-status = 200-399,401,403
max-recursion-depth = 0
exclude-subdirs = %%ff/,.;/,..;/,;/,./,../,%%2e/,%%2e%%2e/
skip-on-status = 429
max-time = 0
target-max-time = 0

[dictionary]
default-extensions = php,asp,aspx,jsp,html,htm
force-extensions = False
wordlist-max-size = 500000

[request]
http-method = get
follow-redirects = False

[connection]
timeout = 7.5
delay = 0
max-retries = 1

[view]
full-url = False
quiet-mode = False
color = True
```

命令行参数始终优先于配置文件。

## 项目结构

```
HiDir/
├── main.go                    # 程序入口
├── config.ini                 # 默认配置文件（与 dirsearch 兼容）
├── assets/                    # 内嵌资源（go:embed）
│   ├── assets.go
│   └── db/                    # 词表数据库（含 dicc.txt、分类词表、黑名单等）
└── internal/
    ├── settings/              # 全局常量与默认配置
    ├── filters/               # 数值范围/大小/时间过滤解析
    ├── utils/                 # 通用工具（URL、编码、爬虫、raw 请求解析等）
    ├── options/               # 命令行解析、配置合并与校验
    ├── wordlist/              # 词表生成引擎（模板展开、分类、运行时字典）
    ├── httpx/                 # HTTP 请求层（认证、代理、限速、响应解析）
    ├── scanner/               # 通配符检测与扫描调度（Fuzzer）
    ├── controller/            # 扫描主控（递归、会话、中断处理）
    ├── report/                # 报告生成（7 种文件格式）
    └── view/                  # 终端输出（颜色、进度条）
```

## 运行测试

```bash
cd HiDir
go test ./...
```

测试覆盖所有命令行参数解析、配置合并、词表生成、过滤逻辑、报告格式、HTTP 认证/重定向/错误分类，以及基于 `httptest` 的端到端扫描集成测试（发现、递归、爬取、跳过状态码、报告写入等）。

## 与 dirsearch 的差异

HiDir 以兼容 dirsearch 的参数与输出为目标，以下为刻意保留或不可避免的差异：

1. **单一 Go 引擎**：`--request-backend` 的 `python`/`native` 均被接受并使用同一个 Go 引擎；`--async`/`--sync` 被接受（Go 天然并发）。
2. **报告格式**：`sqlite`、`mysql`、`postgresql` 格式参数被接受，但当前构建会在运行时提示不支持并跳过（无内置数据库驱动）；其余 7 种文件格式完全支持。
3. **认证**：`basic`、`digest`、`bearer`、`jwt` 完全支持；`ntlm` 参数被接受但运行时提示不支持。
4. **代理**：支持 HTTP/SOCKS5；SOCKS4/4A 在初始化时给出明确错误提示。
5. **词表**：`--wordlist-backend` 参数被接受，Go 版本使用统一的生成实现（与 dirsearch 的 Python 后端语义一致）。
6. **会话文件**：采用 JSON 格式（包含原始参数、剩余队列与输出历史），会话文件本身与 dirsearch 的 JSON 会话不互认。

## 免责声明

本工具仅用于授权的安全测试与安全教育目的。请勿对未获得明确授权的目标使用本工具，使用者需对自身行为负责。
