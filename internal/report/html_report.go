package report

import (
	"encoding/json"
	"html"
	"os"
	"strings"

	"hidir/internal/utils"
)

// htmlTemplate 是 HTML 报告模板，与 dirsearch 的
// html_report_template.html 保持一致（Vue + Bootstrap，支持列排序、
// 状态码/长度排除过滤、搜索与批量打开 URL），
// 仅将展示的 dirsearch 字样替换为 HiDir。
// 占位符：__RESULTS_JSON__、__COMMAND__、__DATE__。
// 模板中 getReadableSize 含反引号，故拆为三段拼接。
const htmlTemplate = htmlTemplatePart1 + "`" + htmlTemplatePart2 + "`" + htmlTemplatePart3

const htmlTemplatePart1 = `<!DOCTYPE html>
<html>
<head>
<meta content="text/html;charset=utf-8" http-equiv="Content-Type">
<meta content="utf-8" http-equiv="encoding">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>HiDir report</title>
<style>
* {
  color: #eef1f3;
}
th.active .arrow {
  opacity: 1;
}
.arrow {
  display: inline-block;
  vertical-align: middle;
  width: 0;
  height: 0;
  margin-left: 5px;
  opacity: 0.66;
}
.arrow.none {
  border-left: 4px solid transparent;
  border-right: 4px solid transparent;
  border-bottom: 4px solid #82b983;
}
.arrow.asc {
  border-left: 4px solid transparent;
  border-right: 4px solid transparent;
  border-bottom: 4px solid #42b983;
}
.arrow.dsc {
  border-left: 4px solid transparent;
  border-right: 4px solid transparent;
  border-top: 4px solid #42b983;
}
</style>
<script src="https://cdn.jsdelivr.net/npm/vue@2.6.12/dist/vue.js" integrity="sha384-ma9ivURrHX5VOB4tNq+UiGNkJoANH4EAJmhxd1mmDq0gKOv88wkKZOfRDOpXynwh" crossorigin="anonymous"></script>
<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.0.0/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-wEmeIV1mKuiNpC+IOBjI7aAzPcEZeedi5yW5f2yOq55WWLwNGmvvx4Um1vskeMj0" crossorigin="anonymous">
<script>

function openURLs(){
    var table = document.getElementsByClassName("table")[0];
    for (let row of table.rows) {
      let val = row.cells[0].innerText;
      if (val == "URL" || val == null)
        continue;
      window.open(val, '_blank');
    }
}
function search(app, result){
    return app.searchQuery.toLowerCase().split(" ").every(
      (v) => result.url.toLowerCase().includes(v) ||
      (result.contentType && result.contentType.toLowerCase().includes(v)) ||
      String(result.contentLength).includes(v) ||
      result.status.toString().includes(v) ||
      (result.redirect && result.redirect.toLowerCase().includes(v))
    );
}
function statusExcludeSearch(app, result){
    return app.statusExcludeSearchQuery.toLowerCase().split(" ").every(
      (v) => !result.status.toString().includes(v)
    );
}
function lengthExcludeSearch(app, result){
    return app.lengthExcludeSearchQuery.toLowerCase().split(" ").every(
      (v) => String(result.contentLength) !== v
    );
}
function sort(app, results){
    if(app.sortColumn == null){
      return results
    }

    return results.sort(function(a, b){
      if(a[app.sortColumn.name] < b[app.sortColumn.name]){
        return -1 * app.sortColumn.order;
      }
      if(a[app.sortColumn.name] > b[app.sortColumn.name]){
        return 1 * app.sortColumn.order;
      }
      return 0
    });
}

window.onload = function () {
    var app = new Vue({
      el: '#app',
      delimiters: ['[[', ']]'],
      data: {
        lengthExcludeSearchQuery: null,
        statusExcludeSearchQuery: null,
        searchQuery: null,
        resources: __RESULTS_JSON__,
        sortColumn: null,
        columns: [
          {name: 'url', text: 'URL', order: -1},
          {name: 'status', text: 'Status', order: -1},
          {name: 'contentLength', text: 'Content Length', order: -1},
          {name: 'contentType', text: 'Content Type', order: -1},
          {name: 'redirect', text: 'Redirect', order: -1},
        ]
      },
      methods: {
        joinUrl: function(url, redirect){
          return new URL(redirect, url).href;
        },
        getReadableSize: function(bytes){
          if (!bytes) return '0 B'

          const base = 1024
          const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB']
          const i = Math.floor(Math.log(bytes) / Math.log(base))

          return `

const htmlTemplatePart2 = `${Math.round(bytes / Math.pow(base, i))} ${units[i]}`

const htmlTemplatePart3 = `
        },
        getStatusColorClass: function(statusCode){
          if (statusCode <= 199) return "";
          if (statusCode <= 299) return "text-success";
          if (statusCode <= 399) return "text-info";
          if (statusCode <= 499) return "text-warning";
          if (statusCode <= 599) return "text-danger";
        }

      },
      computed: {
        results: function(){
          arr = this.resources
          if(this.searchQuery){
            arr = arr.filter((result)=>{
              return search(this, result)
            });
          }
          if(this.statusExcludeSearchQuery){
            arr = arr.filter((result)=>{
              return statusExcludeSearch(this, result)
            });
          }
          if(this.lengthExcludeSearchQuery){
            arr = arr.filter((result)=>{
              return lengthExcludeSearch(this, result)
            });
          }

          return sort(this, arr);
        }
      }
    });
}
</script>
</head>
<body style="background-color: #3f3f3f;">
    <div id="app">
        <div class="panel panel-default">
            <div class="navbar navbar-expand-lg navbar-light bg-light">
                <div class="p-3">
                    <h1><a href="https://github.com/maurosoria/dirsearch" style="text-decoration:none;color:#c84949;">HiDir</a></h1>
                </div>
            </div>
            <br>
            <div class="w-75 p-3 mx-auto">
                <span style="color: #eef1f3;">
                    <strong>Command: </strong>
                        <!-- Prevent Vue.js client-side template injection. Reference: https://portswigger.net/web-security/cross-site-scripting/cheat-sheet#vuejs-reflected -->
                        <code style="display:inline;background-color:#5C6060;color:#F7F9F9;border-radius:4px;">__COMMAND__</code>
                    <br>
                    <strong>Time:</strong> <p style="display:inline;">__DATE__</p>
                    <br>
                </span>
                <br>
                <div class="row">
                    <div class="search-wrapper panel-heading col-sm-4">
                        <input class="form-control" type="text" v-model="statusExcludeSearchQuery" placeholder="Exclude status codes, separated by space" />
                    </div>
                    <div class="search-wrapper panel-heading col-sm-4">
                        <input class="form-control" type="text" v-model="lengthExcludeSearchQuery" placeholder="Exclude content lengths, separated by space" />
                    </div>
                    <div class="col-sm-4">
                      <button onclick="openURLs()" type="button" class="btn btn-danger">Open all URLs</button>
                    </div>
                    <br>
                    <br>
                    <div class="search-wrapper panel-heading col-sm-12">
                        <input class="form-control" type="text" v-model="searchQuery" placeholder="Search" />
                    </div>
                </div>
                <br>
                <div class="table-responsive">
                    <table v-if="resources.length" class="table">
                        <tbody>
                            <tr>
                                <th v-for="column in columns" @click="sortColumn = column; column.order = column.order * -1" :class="{active: column == sortColumn}" style="cursor:context-menu">
                                    [[column.text]]
                                    <span v-bind:class="['arrow', sortColumn != column ? 'none' : column.order > 0 ? 'asc' : 'dsc']"></span>
                                </th>
                            </tr>
                            <tr v-for="result in results">
                                <td><a v-bind:class="getStatusColorClass(result.status)" v-bind:href="result.url" class="text-decoration-none" target="_blank">[[result.url]]</a></td>
                                <td><a v-bind:class="getStatusColorClass(result.status)" class="text-decoration-none" target="_blank">[[result.status]]</a></td>
                                <td><a :title="result.contentLength" target="_blank">[[getReadableSize(result.contentLength)]]</a></td>
                                <td><a target="_blank">[[result.contentType]]</a></td>
                                <!-- If you remove the "style" attribute, the redirect URL will be colored, it's a weird bug that I haven't figured out yet -->
                                <td><a :href="joinUrl(result.url, result.redirect)" target="_blank" style="color: #eef1f3;">[[result.redirect]]</a></td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </div>
</body>
</html>
`

// resourcesPrefix 是 HTML 报告中资源数据行的前缀（与 dirsearch 一致）。
const resourcesPrefix = "        resources: "

// newHTML 生成空的 HTML 报告。
func newHTML(r *fileReport) string {
	return renderHTML(r.meta, []jsonEntry{})
}

// renderHTML 渲染 HTML 报告。
// 命令行先做 HTML 转义、再把 "[" 替换为独立的 <code> 片段，
// 与 dirsearch 模板中 Jinja 的 e | replace 过滤器一致，防止 Vue 模板注入。
func renderHTML(meta StartInfo, results []jsonEntry) string {
	resourcesJSON, _ := json.Marshal(results)
	command := html.EscapeString(meta.Command)
	command = strings.ReplaceAll(command, "[", "<code>[</code>")
	page := strings.ReplaceAll(htmlTemplate, "__RESULTS_JSON__", string(resourcesJSON))
	page = strings.ReplaceAll(page, "__COMMAND__", command)
	page = strings.ReplaceAll(page, "__DATE__", html.EscapeString(meta.StartTime))
	return page
}

// parseHTML 从既有 HTML 报告解析资源数据。
func parseHTML(file string) (any, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, resourcesPrefix) {
			var entries []jsonEntry
			payload := strings.TrimRight(strings.TrimPrefix(line, resourcesPrefix), ",;\n\r \t")
			if err := json.Unmarshal([]byte(payload), &entries); err != nil {
				return nil, err
			}
			return entries, nil
		}
	}
	return nil, errNoResources
}

// errNoResources 表示 HTML 报告缺少资源数据。
var errNoResources = &htmlParseError{}

type htmlParseError struct{}

func (e *htmlParseError) Error() string { return "HTML report does not contain resources data" }

// saveHTML 追加结果并整体重写。
func saveHTML(r *fileReport, file string, result Result) {
	data, err := os.ReadFile(file)
	if err != nil {
		return
	}
	var entries []jsonEntry
	found := false
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, resourcesPrefix) {
			payload := strings.TrimRight(strings.TrimPrefix(line, resourcesPrefix), ",;\n\r \t")
			if json.Unmarshal([]byte(payload), &entries) != nil {
				return
			}
			found = true
			break
		}
	}
	if !found {
		return
	}
	entry := jsonEntry{
		URL:           result.URL,
		Status:        result.Status,
		ContentLength: result.Length,
		ContentType:   result.Type,
		Redirect:      result.Redirect,
	}
	if ElapsedMS(result.Elapsed) > 0 {
		entry.Elapsed = result.Elapsed
	}
	entries = append(entries, entry)
	_ = utils.AtomicWriteText(file, renderHTML(r.meta, entries))
}
