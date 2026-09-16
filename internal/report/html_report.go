package report

import (
	"encoding/json"
	"html"
	"os"
	"strings"

	"hidir/internal/utils"
)

// htmlTemplate 是 HTML 报告模板（结构兼容 dirsearch 的
// html_report_template.html，资源数据行保持相同格式以便互操作）。
// 占位符：__RESULTS_JSON__、__COMMAND__、__DATE__。
const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
<meta content="text/html;charset=utf-8" http-equiv="Content-Type">
<meta content="utf-8" http-equiv="encoding">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>dirsearch report</title>
<style>
* { color: #eef1f3; }
body { background-color: #3f3f3f; font-family: sans-serif; }
table { border-collapse: collapse; }
th, td { text-align: left; padding: 6px 12px; }
th { cursor: pointer; }
a { text-decoration: none; }
.status-2 { color: #82b983; }
.status-3 { color: #61afef; }
.status-4 { color: #e5c07b; }
.status-5 { color: #e06c75; }
.search { background-color: #5C6060; color: #F7F9F9; border: none; border-radius: 4px; padding: 6px 10px; margin: 4px 0; }
code { display:inline; background-color:#5C6060; border-radius:4px; }
</style>
<script>
function getReadableSize(bytes) {
  if (!bytes) return '0 B';
  var base = 1024;
  var units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
  var i = Math.floor(Math.log(bytes) / Math.log(base));
  return Math.round(bytes / Math.pow(base, i)) + ' ' + units[i];
}
function statusClass(status) {
  if (status <= 199) return '';
  if (status <= 299) return 'status-2';
  if (status <= 399) return 'status-3';
  if (status <= 499) return 'status-4';
  return 'status-5';
}
var resources = [];
var reportData = {
        resources: __RESULTS_JSON__,
        empty: true
};
function render() {
  var body = document.getElementById('rows');
  body.innerHTML = '';
  var query = (document.getElementById('search').value || '').toLowerCase();
  for (var i = 0; i < resources.length; i++) {
    var result = resources[i];
    if (query &&
        result.url.toLowerCase().indexOf(query) < 0 &&
        (result.contentType || '').toLowerCase().indexOf(query) < 0 &&
        String(result.contentLength).indexOf(query) < 0 &&
        String(result.status).indexOf(query) < 0 &&
        (result.redirect || '').toLowerCase().indexOf(query) < 0) {
      continue;
    }
    var tr = document.createElement('tr');
    tr.innerHTML =
      '<td><a class="' + statusClass(result.status) + '" href="' + result.url + '" target="_blank">' + result.url + '</a></td>' +
      '<td>' + result.status + '</td>' +
      '<td title="' + result.contentLength + '">' + getReadableSize(result.contentLength) + '</td>' +
      '<td>' + (result.contentType || '') + '</td>' +
      '<td>' + (result.redirect || '') + '</td>';
    body.appendChild(tr);
  }
}
window.onload = function() {
  resources = reportData.resources;
  document.getElementById('search').oninput = render;
  render();
};
</script>
</head>
<body>
    <h1>dirsearch report</h1>
    <span>
        <strong>Command: </strong>
        <code>__COMMAND__</code>
        <br>
        <strong>Time:</strong> <span>__DATE__</span>
        <br>
    </span>
    <br>
    <input class="search" id="search" type="text" placeholder="Search" />
    <table>
        <thead>
            <tr>
                <th>URL</th>
                <th>Status</th>
                <th>Content Length</th>
                <th>Content Type</th>
                <th>Redirect</th>
            </tr>
        </thead>
        <tbody id="rows"></tbody>
    </table>
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
func renderHTML(meta StartInfo, results []jsonEntry) string {
	resourcesJSON, _ := json.Marshal(results)
	page := strings.ReplaceAll(htmlTemplate, "__RESULTS_JSON__", string(resourcesJSON))
	page = strings.ReplaceAll(page, "__COMMAND__", html.EscapeString(meta.Command))
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
