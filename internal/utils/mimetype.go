package utils

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"regexp"
	"strings"
)

// queryStringRegex 与 dirsearch 的 QUERY_STRING_REGEX 一致，
// 用于识别 a=b&c=d 形式的查询串。
var queryStringRegex = regexp.MustCompile(`^(\&?([^=& ]+)\=([^=& ]+)?){1,200}$`)

// GuessMimetype 根据请求体内容猜测 Content-Type：
// JSON -> XML -> 查询串 -> 纯文本。
func GuessMimetype(content []byte) string {
	text := string(content)
	switch {
	case isJSON(text):
		return "application/json"
	case isXML(text):
		return "application/xml"
	case queryStringRegex.MatchString(text):
		return "application/x-www-form-urlencoded"
	default:
		return "text/plain"
	}
}

// isJSON 判断内容是否为合法 JSON。
func isJSON(content string) bool {
	var v any
	return json.Unmarshal([]byte(content), &v) == nil
}

// isXML 判断内容是否为合法 XML（须包含至少一个元素节点）。
func isXML(content string) bool {
	decoder := xml.NewDecoder(strings.NewReader(content))
	sawElement := false
	// 循环消费所有 token；干净到达末尾（io.EOF）且出现过元素即为合法 XML
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return sawElement
		}
		if err != nil {
			return false
		}
		if _, ok := token.(xml.StartElement); ok {
			sawElement = true
		}
	}
}
