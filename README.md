<div align="center">

# HiDir

**高性能 Web 路径（目录 / 文件）扫描器 —— dirsearch 的 Go 实现**

单文件运行 · 零依赖 · 完整兼容 dirsearch 命令行参数 · 内置全套词表数据库

[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Release](https://img.shields.io/github/v/release/hicode0101/HiDir?color=181717&logo=github)](https://github.com/hicode0101/HiDir/releases)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-blue)](#install)
[![License](https://img.shields.io/badge/License-GPL--2.0-red)](#license)

[功能特性](#features) · [快速开始](#quick-start) · [参数手册](#parameters) · [报告输出](#reports) · [暂停与恢复](#sessions) · [配置文件](#config)

</div>

---

<a id="features"></a>
## ✨ 功能特性

- **单文件开箱即用**：Go 编译为单个可执行文件，dirsearch 的完整词表数据库（`dicc.txt`、30+ 分类词表、状态码黑名单、User-Agent 列表）通过 `go:embed` 嵌入，无需 Python 环境与任何第三方依赖
- **多种目标输入**：单个/多个 URL、URL 列表文件、标准输入、CIDR 网段、Burp 原始请求文件、Nmap XML 报告、历史会话恢复
- **强大的词表引擎**：`%EXT%` 等 15+ 模板占位符、强制/覆盖/排除扩展名、前后缀注入、大小写变换、多词表合并去重、条目数上限保护
- **智能降噪**：软 404 通配符自动检测（随机采样 + 动态 diff + 重定向正则）、自动校准、重复响应指纹过滤、状态码黑名单词表
- **ffuf 风格高级匹配/过滤**：状态码、长度、词数、行数、正则、响应头、耗时，支持 `and` / `or` 组合
- **递归与爬取**：普通/逐层/强制递归、深度与状态码控制、子目录扫描、响应爬取、备份文件探测
- **完整请求能力**：全 HTTP 方法、请求体、自定义头、Cookie、Basic/Digest/Bearer/JWT 认证、客户端证书、随机 User-Agent
- **连接控制**：HTTP/SOCKS5 代理、Tor、重放代理、限速、请求间隔、失败重试、`--ip` 直连
- **8 种报告格式**：simple / plain / json / xml / md / csv / **html（可排序、可搜索、可过滤的交互式报告）** / **sqlite**
- **断点续扫**：Ctrl+C 暂停菜单（继续 / 下一目录 / 跳过目标 / 保存退出），会话保存与恢复，再次 Ctrl+C 立即强退
- **彩色终端**：按状态码着色、实时进度条、quiet / disable-cli / verbose 三种输出模式

<a id="quick-start"></a>
## 🚀 快速开始

```bash
# 最简单的扫描（使用内置词表 dicc.txt）
hidir -u https://example.com/

# 指定扩展名 + 50 线程 + 递归
hidir -u https://example.com/ -e php,html,js -t 50 -r

# 使用全部常见扩展名，强制为每个词条追加
hidir -u https://example.com/ -e '*' -f

# 多目标 + JSON / HTML 双格式报告
hidir -l urls.txt -e php -O json,html -o 'reports/{host}-{date}.{extension}'

# 带认证、走 Tor、限速扫描
hidir -u https://example.com/ -e php --auth "admin:pass" --auth-type basic --tor --max-rate 50
```

<a id="install"></a>
## 📦 安装与构建

**方式一：下载预编译包（推荐）**

前往 [Releases](https://github.com/hicode0101/HiDir/releases) 下载对应平台的压缩包，解压即用，无需安装 Go：

| 文件 | 平台 |
|---|---|
| `hidir_<版本>_windows_amd64.zip` | Windows 64 位 |
| `hidir_<版本>_linux_amd64.tar.gz` | Linux 64 位 |
| `hidir_<版本>_darwin_amd64.tar.gz` | macOS（Intel 芯片） |
| `hidir_<版本>_darwin_arm64.tar.gz` | macOS（Apple Silicon M 系列） |

每个包都附 SHA256 校验和（汇总见 `checksums.txt`）。推送 `v*` 标签时由 GitHub Actions 自动构建发布。

**方式二：从源码构建**

需要 Go 1.25+：

```bash
git clone https://github.com/hicode0101/HiDir.git
cd HiDir
go build -o hidir .        # Windows 下为 hidir.exe
go test ./...              # 运行全部测试
```

产物为单个可执行文件，词表数据库已嵌入，可直接拷贝到其他机器使用。两个可选环境变量：

| 环境变量 | 作用 |
|---|---|
| `HIDIR_DB` | 指定外部词表目录（覆盖内嵌 `db/`），便于自定义词表 |
| `HIDIR_CONFIG` | 指定默认配置文件路径（兼容 `DIRSEARCH_CONFIG`） |

若工作目录或可执行文件同目录存在 `db/` 目录，会优先使用磁盘上的词表文件。

<a id="parameters"></a>
## 📖 参数手册

> **阅读约定**：除「目标输入」组外，示例均省略公共前缀 `hidir -u https://example.com/`，拼接即可执行。状态码类参数均支持范围写法（如 `200,300-399`），大小支持 `1024` / `1KB` 等写法。

### 目标输入（Mandatory）

| 参数 | 说明 | 示例 |
|---|---|---|
| `-u, --url=URL` | 目标 URL，可重复传入；无 scheme 时自动探测 http/https | `hidir -u https://a.com/ -u http://b.com/` |
| `-l, --urls-file=PATH` | URL 列表文件，`#` 开头为注释 | `hidir -l urls.txt -e php` |
| `--stdin` | 从标准输入读取 URL，可配合管道 | `cat urls.txt \| hidir --stdin -e php` |
| `--cidr=CIDR` | 将 CIDR 网段展开为扫描目标 | `hidir --cidr 192.168.1.0/24 -e php` |
| `--raw=PATH` | 从原始 HTTP 请求文件还原目标、方法、头与请求体（Burp 抓包） | `hidir --raw request.txt --scheme https` |
| `--nmap-report=PATH` | 从 nmap XML 报告提取目标（建议 nmap 扫描时加 `-sV`） | `hidir --nmap-report scan.xml -e php` |
| `-s, --session=FILE` | 恢复会话文件继续扫描 | `hidir -s sessions/session_xxx.json` |
| `--session-id=ID` | 按编号恢复会话，编号来自 `--list-sessions` | `hidir --session-id 2` |
| `--config=PATH` | 指定配置文件（默认：`HIDIR_CONFIG` → `DIRSEARCH_CONFIG` → 当前目录 `config.ini`） | `hidir -u https://a.com/ --config my.ini` |

### 词表设置（Dictionary Settings）

| 参数 | 说明 | 示例 |
|---|---|---|
| `-w, --wordlists=PATHS` | 词表文件或目录，逗号分隔；缺省使用内置 `dicc.txt`（约 9700 条） | `-w db/dicc.txt,extra.txt` |
| `--wordlist-categories=NAMES` | 内置分类词表：`all` 为全部，`php/*` 为前缀通配 | `--wordlist-categories common,web` |
| `--wordlist-backend=BACKEND` | 词表生成后端：`auto`、`python`、`native`（Go 版等价，保留兼容，默认 `auto`） | `--wordlist-backend native` |
| `--wordlist-status` | 只显示词表解析结果与条目数，不发起扫描 | `--wordlist-status -e php` |
| `--wordlist-max-size=COUNT` | 生成词表的条目数上限，超出报错（默认 500000） | `--wordlist-max-size 100000` |
| `-e, --extensions=EXTS` | 扩展名列表；`'*'` 为常见扩展名集合（php,jsp,asp,aspx,do,action,cgi,html,htm,js,tar.gz） | `-e php,html` 或 `-e '*'` |
| `-f, --force-extensions` | 为**每个**词条追加扩展名（默认仅替换词条中的 `%EXT%` 占位符） | `-e php -f` |
| `--overwrite-extensions` | 用 `-e` 选定的扩展名覆盖词条自带的扩展名 | `-e php --overwrite-extensions` |
| `--exclude-extensions=EXTS` | 排除的扩展名（作用于爬取、备份探测等动态路径） | `--exclude-extensions asp,jsp` |
| `--prefixes=PREFIXES` | 为所有词条添加前缀 | `--prefixes .,_` |
| `--suffixes=SUFFIXES` | 为所有非目录词条添加后缀 | `--suffixes ~,.bak` |
| `-U, --uppercase` | 词表全部大写（`admin` → `ADMIN`） | `-w words.txt -U` |
| `-L, --lowercase` | 词表全部小写 | `-w words.txt -L` |
| `-C, --capital` | 词表首字母大写（`admin` → `Admin`） | `-w words.txt -C` |

<details>
<summary><b>词表模板占位符（点击展开）</b></summary>

词表中可使用以下占位符，扫描时自动展开：

| 占位符 | 展开结果 |
|---|---|
| `%EXT%` | 替换为 `-e` 指定的扩展名（未指定时使用默认扩展名） |
| `%SUBJECT%` | 目标主机名 |
| `%CRUD_OP%` / `%AUTH_OP%` / `%ADMIN_OP%` | 常见操作词（create/read/update/delete、login/register/logout、add/edit/del 等） |
| `%ENV%` | dev / test / prod / staging 等 |
| `%DB%` | mysql / postgres / redis / mongodb 等 |
| `%ARCHIVE%` | zip / tar / tar.gz / tgz / gz / 7z / rar / bak |
| `%BACKUP%` | bak / backup / old / orig / save 等 |
| `%API_VERSION%` | v1 / v2 / v3 / api/v1 等 |
| `%YYYY%` / `%MM%` / `%DD%` / `%DATE%` | 年 / 月 / 日 / 完整日期 |

</details>

### 通用设置（General Settings）

| 参数 | 说明 | 示例 |
|---|---|---|
| `-t, --threads=THREADS` | 并发线程数（默认 25） | `-t 50` |
| `-r, --recursive` | 递归扫描发现的目录 | `-r` |
| `--deep-recursive` | 对发现的每一层目录深度都递归（`api/users` → `api/` 也入队） | `--deep-recursive` |
| `--force-recursive` | 对发现的每个路径递归，不限于目录 | `--force-recursive` |
| `-R, --max-recursion-depth=DEPTH` | 最大递归深度，0 为无限（默认 0） | `-R 3` |
| `--recursion-status=CODES` | 触发递归的状态码（默认 `200-399,401,403`） | `--recursion-status 200-299,401` |
| `--subdirs=SUBDIRS` | 扫描目标 URL 的指定子目录 | `--subdirs api/,admin/` |
| `--exclude-subdirs=SUBDIRS` | 递归时排除的子目录 | `--exclude-subdirs /static,/uploads` |
| `-i, --include-status=CODES` | 仅保留这些状态码的结果 | `-i 200,300-399` |
| `-x, --exclude-status=CODES` | 排除这些状态码的结果 | `-x 301,500-599` |
| `--exclude-sizes=SIZES` | 按响应大小排除 | `--exclude-sizes 0,0B,4KB` |
| `--exclude-text=TEXTS` | 按响应文本排除，可重复传入 | `--exclude-text "Not Found" --exclude-text "错误"` |
| `--exclude-regex=REGEX` | 按正则排除响应 | `--exclude-regex "^404$"` |
| `--exclude-redirect=STRING` | 跳转地址匹配该正则/文本时排除 | `--exclude-redirect "/index\.html"` |
| `--exclude-response=PATH` | 排除与指定页面相似的响应（如本地 404 页面样本） | `--exclude-response 404.html` |
| `--skip-on-status=CODES` | 命中这些状态码时跳过整个目标（默认 `429`） | `--skip-on-status 429,503` |
| `--min-response-size=LENGTH` | 最小响应长度，小于则过滤 | `--min-response-size 1KB` |
| `--max-response-size=LENGTH` | 最大响应长度，大于则过滤 | `--max-response-size 100KB` |
| `--filter-threshold=THRESHOLD` | 相同响应重复出现超过该次数后开始过滤 | `--filter-threshold 30` |
| `--max-time=SECONDS` | 整个扫描的最长运行时间 | `--max-time 3600` |
| `--target-max-time=SECONDS` | 单个目标的最长运行时间 | `--target-max-time 600` |
| `--exit-on-error` | 请求出错时立即退出 | `--exit-on-error` |
| `--list-sessions` | 列出可恢复的会话后退出 | `hidir --list-sessions` |
| `--sessions-dir=PATH` | 会话存储目录（默认 `~/.hidir/sessions`） | `--sessions-dir D:\sessions` |
| `-a, --async` / `--sync, --no-async` | 异步/同步模式开关（Go 天然并发，仅为兼容 dirsearch 保留） | `-a` |

### 高级过滤（Advanced Filtering）

> 与 ffuf/wfuzz 对齐的匹配器（match，保留命中）与过滤器（filter，剔除命中）体系，`--mmode` / `--fmode` 控制同组多个条件的组合方式（默认 `or`）。

| 参数 | 说明 | 示例 |
|---|---|---|
| `--auto-calibration` | 强制从扫描一开始就进行通配符校准 | `--auto-calibration` |
| `--matcher-mode, --mmode=MODE` | 匹配器组合方式：`and`、`or`（默认 `or`） | `--mc 200 --ms 100-5000 --mmode and` |
| `--filter-mode, --fmode=MODE` | 过滤器组合方式：`and`、`or`（默认 `or`） | `--fc 404 --fs 0 --fmode or` |
| `--match-status, --mc=CODES` | 状态码匹配器 | `--mc 200,204,301` |
| `--filter-status, --fc=CODES` | 状态码过滤器 | `--fc 404,410` |
| `--match-size, --ms=SIZES` | 响应长度匹配器 | `--ms 100-5000` |
| `--filter-size, --fs=SIZES` | 响应长度过滤器 | `--fs 0` |
| `--match-words, --mw=WORDS` | 响应词数匹配器 | `--mw '>100'` |
| `--filter-words, --fw=WORDS` | 响应词数过滤器 | `--fw '<10'` |
| `--match-lines, --ml=LINES` | 响应行数匹配器 | `--ml 10-50` |
| `--filter-lines, --fl=LINES` | 响应行数过滤器 | `--fl '>200'` |
| `--match-regex, --mr=REGEX` | 响应体正则匹配器 | `--mr "admin_panel"` |
| `--filter-regex, --fr=REGEX` | 响应体正则过滤器 | `--fr "maintenance"` |
| `--match-header=TEXT` | 响应头文本匹配器，可重复传入 | `--match-header "nginx"` |
| `--filter-header=TEXT` | 响应头文本过滤器，可重复传入 | `--filter-header "cloudflare"` |
| `--match-header-regex=REGEX` | 响应头正则匹配器 | `--match-header-regex "Content-Type: application/json"` |
| `--filter-header-regex=REGEX` | 响应头正则过滤器 | `--filter-header-regex "Set-Cookie: .*Session"` |
| `--match-time, --mt=TIME` | 响应耗时匹配器（毫秒） | `--mt '>500'` |
| `--filter-time, --ft=TIME` | 响应耗时过滤器（毫秒） | `--ft '>2000'` |

### 请求设置（Request Settings）

| 参数 | 说明 | 示例 |
|---|---|---|
| `-m, --http-method=METHOD` | HTTP 方法（默认 `GET`） | `-m POST` |
| `--request-backend=BACKEND` | 请求后端：`python`、`native`（Go 版两者等价，默认 `python`） | `--request-backend native` |
| `-d, --data=DATA` | HTTP 请求体 | `-d "username=admin&password=admin"` |
| `--data-file=PATH` | 从文件读取请求体（不转义、不转换换行，适合 JSON） | `--data-file payload.json` |
| `-H, --header=HEADER` | 自定义请求头，可重复传入 | `-H "X-Forwarded-For: 127.0.0.1"` |
| `--headers-file=PATH` | 从文件读取请求头 | `--headers-file headers.txt` |
| `-F, --follow-redirects` | 跟随 HTTP 重定向 | `-F` |
| `--random-agent` | 每个请求随机选择 User-Agent（内置列表） | `--random-agent` |
| `--user-agent=UA` | 自定义 User-Agent | `--user-agent "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"` |
| `--cookie=COOKIE` | Cookie | `--cookie "session=abc123; token=xyz"` |
| `--auth=CREDENTIAL` | 认证凭据，配合 `--auth-type` | 见下 |
| `--auth-type=TYPE` | 认证类型：`basic`、`digest`、`bearer`、`jwt`（`ntlm` 参数被接受但当前构建不支持） | `--auth "admin:pass" --auth-type basic` |
| `--cert-file=PATH` | 客户端证书文件 | `--cert-file client.pem --key-file client.key` |
| `--key-file=PATH` | 客户端证书私钥文件（未加密） | 同上 |

认证类型示例：

```bash
--auth "admin:123456" --auth-type basic      # Basic
--auth "admin:123456" --auth-type digest     # Digest
--auth "eyJhbGciOi..." --auth-type bearer    # Bearer Token
--auth "eyJhbGciOi..." --auth-type jwt       # JWT
```

### 连接设置（Connection Settings）

| 参数 | 说明 | 示例 |
|---|---|---|
| `--timeout=SECONDS` | 连接超时秒数（默认 7.5） | `--timeout 15` |
| `--delay=SECONDS` | 请求间隔秒数（默认 0） | `--delay 0.5` |
| `-p, --proxy=PROXY` | 代理 URL（HTTP/SOCKS5），可重复传入实现轮询 | `-p http://127.0.0.1:8080` |
| `--proxies-file=PATH` | 从文件读取代理列表 | `--proxies-file proxies.txt` |
| `--proxy-auth=CREDENTIAL` | 代理认证凭据 | `--proxy-auth user:pass` |
| `--replay-proxy=PROXY` | 发现的路径经该代理重放（如 Burp） | `--replay-proxy http://127.0.0.1:8080` |
| `--tor` | 使用 Tor 作为代理（SOCKS5 `127.0.0.1:9050`/`9150`） | `--tor` |
| `--scheme=SCHEME` | 指定协议（默认对 443 端口做 TLS 握手自动探测） | `--scheme https` |
| `--max-rate=RATE` | 每秒最大请求数，0 为不限（默认 0） | `--max-rate 100` |
| `--retries=RETRIES` | 失败重试次数（默认 1） | `--retries 3` |
| `--ip=IP` | 直连指定服务器 IP（绕过 DNS） | `--ip 93.184.216.34` |
| `--interface=IFACE` | 绑定网络接口（仅 Linux） | `--interface eth0` |

### 进阶设置（Advanced Settings）

| 参数 | 说明 | 示例 |
|---|---|---|
| `--crawl` | 爬取响应中的链接并加入扫描队列（支持 HTML / 文本 / robots.txt） | `--crawl` |
| `--find-backup` | 探测已发现文件的备份（`admin.php` → `admin.php.bak`、`admin.php~`、`admin.zip` 等） | `--find-backup` |

### 视图设置（View Settings）

| 参数 | 说明 | 示例 |
|---|---|---|
| `--full-url` | 结果输出完整 URL（quiet 模式自动启用） | `--full-url` |
| `--redirects-history` | 显示重定向链路历史 | `--redirects-history` |
| `--no-color` | 关闭彩色输出 | `--no-color` |
| `-q, --quiet-mode` | 安静模式，只输出命中结果（便于管道处理） | `-q --full-url \| grep 200` |
| `--disable-cli` | 关闭全部命令行输出（仅写报告） | `--disable-cli` |
| `-v, --verbose` | 显示响应耗时与 Content-Type | `-v` |

### 输出设置（Output Settings）

| 参数 | 说明 | 示例 |
|---|---|---|
| `-O, --output-formats=FORMAT` | 报告格式，逗号分隔：`simple`、`plain`、`json`、`xml`、`md`、`csv`、`html`、`sqlite` | `-O json,html` |
| `-o, --output-file=PATH` | 输出文件路径，支持路径变量（见下节） | `-o 'reports/{host}-{date}.{extension}'` |
| `--mysql-url=URL` | MySQL 输出地址（参数被接受，当前构建运行时提示不支持） | `--mysql-url mysql://user:pass@host/db` |
| `--postgres-url=URL` | PostgreSQL 输出地址（同上） | `--postgres-url postgres://user:pass@host/db` |
| `--save-response=PATH` | 将命中的响应体保存到目录（单个最大 80 MiB） | `--save-response responses/` |
| `--save-response-jsonl=PATH` | 将命中的响应追加到 JSONL 文件（Body 为 Base64） | `--save-response-jsonl hits.jsonl` |
| `--log=PATH` | 日志文件 | `--log hidir.log` |

### 帮助

| 参数 | 说明 |
|---|---|
| `-h, --help` | 显示常用选项帮助 |
| `-hh, --help-all` | 显示全部选项帮助 |
| `--version` | 显示版本号 |

<a id="reports"></a>
## 📊 报告输出

支持 8 种格式，可任意组合：

| 格式 | 扩展名 | 说明 |
|---|---|---|
| `simple` | `.txt` | 每行一个 URL，最精简 |
| `plain` | `.txt` | `状态码 大小 URL (耗时) [-> 跳转]`，默认格式 |
| `json` | `.json` | 结构化 JSON（含命令行与时间元数据） |
| `xml` | `.xml` | `<hidirscan>` 根元素的结构化 XML |
| `md` | `.md` | Markdown 表格 |
| `csv` | `.csv` | 标准逗号分隔表格 |
| `html` | `.html` | **交互式单页报告**（见下） |
| `sqlite` | `.sqlite` | SQLite 数据库，表结构与 dirsearch 一致，表名默认取自 `{scheme}_{host}:{port}` 模板（配置项 `output-sql-table`） |

**HTML 交互式报告**内置排序、搜索与过滤能力：

- 点击任意列头排序（升/降序箭头指示）
- 全文搜索：同时匹配 URL、状态码、大小、Content-Type、跳转地址
- 按状态码排除（如输入 `404 301`）、按内容长度排除
- **Open all URLs** 一键批量打开全部命中链接
- 按状态码着色（2xx 绿 / 3xx 青 / 4xx 黄 / 5xx 红），可读文件大小显示

**路径变量**：`-o` 与 `output-sql-table` 均支持 `{format}`、`{extension}`、`{host}`、`{scheme}`、`{port}`、`{date}`（DD-MM-YYYY）、`{datetime}`（DD-MM-YYYY_HH-MM-SS）。

多格式输出到同一 `-o` 时，路径必须同时包含 `{format}` 与 `{extension}` 变量以区分文件：

```bash
hidir -u https://example.com/ -e php -O simple,json,html,sqlite -o 'out/report.{format}.{extension}'
# 生成 out/report.simple.txt / report.json.json / report.html.html / report.sql.sqlite
```

报告文件具备断点续写能力：中断后恢复会话，结果会追加进同一份报告。

<a id="sessions"></a>
## ⏸️ 暂停与会话恢复

扫描过程中按 `Ctrl+C`，扫描线程立即暂停并显示交互菜单（菜单期间进度条停止刷新，再次按 `Ctrl+C` 立即强制退出）：

```text
[q]uit / [c]ontinue / [n]ext / [s]kip target:
```

| 选项 | 作用 |
|---|---|
| `[c]ontinue` | 继续扫描 |
| `[n]ext` | 跳到下一个目录任务（多目录时出现） |
| `[s]kip` | 跳过当前目标（多目标时出现） |
| `[q]uit` | 退出，可再选 `[s]ave` 保存会话 / `[q]uit` 不保存 |

保存的会话包含原始命令行参数、剩余目标/目录队列、计数器与完整输出历史，恢复时自动回放历史输出：

```bash
hidir --list-sessions                                   # 列出可恢复会话
hidir --session-id 2                                    # 按编号恢复
hidir -s ~/.hidir/sessions/2026-09-17/session_xxx.json  # 按文件恢复
```

<a id="config"></a>
## ⚙️ 配置文件

查找顺序：`--config` 参数 → `HIDIR_CONFIG` → `DIRSEARCH_CONFIG` 环境变量 → 当前目录 `config.ini` → 可执行文件同目录 `config.ini` → 内置默认配置。**命令行参数始终优先于配置文件**。

```ini
[general]
threads = 25
async = True
recursive = False
deep-recursive = False
force-recursive = False
recursion-status = 200-399,401,403
max-recursion-depth = 0
exclude-subdirs = %%ff/,.;/,..;/,;/,./,../,%%2e/,%%2e%%2e/
random-user-agents = False
max-time = 0
target-max-time = 0
exit-on-error = False
skip-on-status = 429
auto-calibration = False

[advanced-filtering]
matcher-mode = or
filter-mode = or

[dictionary]
default-extensions = php,asp,aspx,jsp,html,htm
force-extensions = False
overwrite-extensions = False
lowercase = False
uppercase = False
capital = False
wordlist-backend = auto
wordlist-max-size = 500000

[request]
http-method = get
request-backend = python
follow-redirects = False

[connection]
timeout = 7.5
delay = 0
max-rate = 0
max-retries = 1

[advanced]
crawl = False
find-backup = False

[view]
full-url = False
quiet-mode = False
color = True
show-redirects-history = False
disable-cli = False
verbose = False

[output]
output-formats = plain
output-sql-table = {scheme}_{host}:{port}
```

## 🧭 使用示例

```bash
# 软 404 站点降噪：排除 0 字节与 4KB 响应 + 文本过滤
hidir -u https://example.com/ -e php -i 200-399 --exclude-sizes 0,4KB --exclude-text "not found"

# ffuf 风格：状态 200 且长度在 100-5000 之间才显示
hidir -u https://example.com/ -e '*' --mc 200 --ms 100-5000 --mmode and

# 递归扫描最多 3 层，逐层递归，排除静态目录
hidir -u https://example.com/ -e php -r -R 3 --deep-recursive --exclude-subdirs /static,/assets

# 爬取 + 备份探测组合拳
hidir -u https://example.com/ -e php --crawl --find-backup

# POST 登录表单扫描（带 Cookie 与自定义头）
hidir -u https://example.com/ -m POST -d "user=admin&pass=admin" \
  -H "X-Forwarded-For: 127.0.0.1" --cookie "session=abc123"

# 走 Burp 代理扫描，命中的路径再重放一遍
hidir -u https://example.com/ -e php -p http://127.0.0.1:8080 --replay-proxy http://127.0.0.1:8080

# 整个 C 段 + 限速 + 随机 UA + 最长 1 小时
hidir --cidr 192.168.1.0/24 -e php -t 100 --max-rate 200 --random-agent --max-time 3600

# 安静模式管道处理：只输出 200 的完整 URL
hidir -u https://example.com/ -e php -q --full-url | grep " 200 "

# 从 Burp 原始请求 / nmap 报告获取目标
hidir --raw request.txt --scheme https
hidir --nmap-report nmap_scan.xml -e php
```

## 🆚 与 dirsearch 的差异

HiDir 以参数与输出完全兼容 dirsearch 为目标，以下为刻意保留或不可避免的差异：

1. **单引擎**：Go 天然并发，`--request-backend` 的 `python`/`native`、`--async`/`--sync` 均被接受且等价。
2. **数据库格式**：`sqlite` 已原生支持（纯 Go 驱动，无 CGO）；`mysql`、`postgresql` 参数被接受但运行时提示不支持。
3. **认证**：`basic`、`digest`、`bearer`、`jwt` 完全支持；`ntlm` 参数被接受但运行时提示不支持。
4. **词表后端**：`--wordlist-backend` 参数被接受，Go 版使用统一实现，语义与 dirsearch 一致。
5. **会话文件**：采用自有 JSON 格式（参数、剩余队列、输出历史），与 dirsearch 的会话文件不互认。
6. **报告前缀**：plain 报告头与 XML 根元素使用 `HiDir` / `hidirscan` 标识，其余字段与 dirsearch 逐字段对齐。

## 🧪 开发与测试

```bash
go build -o hidir .   # 构建
go test ./...         # 全量测试（含基于 httptest 的端到端扫描集成测试）
go vet ./...          # 静态检查
```

发布流程：推送 `v*` 标签（如 `git tag v1.0.1 && git push origin v1.0.1`）后，
GitHub Actions 自动交叉编译 Windows / Linux / macOS（Intel 与 Apple Silicon）
四类产物并创建 Release，版本号在编译时注入（`hidir --version` 与启动横幅）。

测试覆盖：参数解析与配置合并、词表生成（模板/分类/上限）、过滤逻辑、8 种报告格式（含 sqlite 读写与文件冲突检测）、HTTP 认证/重定向/错误分类，以及扫描发现、递归、爬取、跳过状态码、会话保存与恢复等端到端流程。

<details>
<summary><b>项目结构</b></summary>

```text
HiDir/
├── main.go                    # 程序入口
├── config.ini                 # 默认配置文件（与 dirsearch 兼容）
├── assets/                    # 内嵌资源（go:embed）
│   ├── assets.go
│   └── db/                    # 词表数据库：dicc.txt、分类词表、黑名单、User-Agent
└── internal/
    ├── settings/              # 全局常量与默认值
    ├── filters/               # 数值范围/大小/时间过滤解析
    ├── utils/                 # 通用工具（URL、编码、爬虫、raw 请求解析）
    ├── options/               # 命令行解析、配置合并与校验
    ├── wordlist/              # 词表生成引擎（模板展开、分类、运行时字典）
    ├── httpx/                 # HTTP 请求层（认证、代理、限速、响应解析）
    ├── scanner/               # 通配符检测与扫描调度（Fuzzer）
    ├── controller/            # 扫描主控（递归、会话、Ctrl+C 中断处理）
    ├── report/                # 报告生成（8 种格式，含 HTML 模板与 sqlite）
    └── view/                  # 终端输出（颜色、进度条）
```

</details>

## 🤝 参与贡献

欢迎 Issue 与 Pull Request：

1. 提交 Bug 请附完整命令行、目标特征（脱敏）与复现步骤
2. 新功能建议先开 Issue 讨论，保持与 dirsearch 的兼容性
3. 提交前请确保 `go test ./...` 与 `go vet ./...` 通过

<a id="license"></a>
## 📜 许可证

本项目基于 [GPL-2.0](https://www.gnu.org/licenses/old-licenses/gpl-2.0.html) 发布，继承自 dirsearch 的许可证条款。任何再分发需遵循 GPL-2.0 并保留原版权声明。

## 🙏 致谢

- [dirsearch](https://github.com/maurosoria/dirsearch) —— Mauro Soria 与社区贡献者的杰出作品，HiDir 的词表数据库、参数体系与交互设计均源自该项目

## ⚠️ 免责声明

本工具仅用于**授权的安全测试**与安全教育目的。请勿对未获得明确书面授权的目标使用本工具，因滥用本工具造成的任何后果由使用者自行承担。
