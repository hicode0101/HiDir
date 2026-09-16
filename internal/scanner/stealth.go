// Package scanner 实现通配符（软 404）检测与扫描调度，
// 对应 dirsearch 的 lib/core/scanner.py、fuzzer.py 与 lib/utils/random.py。
package scanner

import (
	"math"
	"math/rand"
	"strings"
)

// stealthSeparators 是隐匿词的分隔符集合。
var stealthSeparators = []string{"-", "_"}

// commonDirectories 是不能出现在隐匿词中的常见目录名。
var commonDirectories = map[string]bool{
	"admin": true, "backup": true, "api": true, "test": true,
}

// stealthWordBank 是隐匿词词库（与 dirsearch 一致）。
var stealthWordBank = []string{
	"cyberfluxion", "luminastra", "aerolithic", "peltorian",
	"chronometral", "syncorance", "neosphereic", "cryptogenoid",
	"polysystive", "megastatary", "hyperlogism", "omnitechant",
	"exovalable", "tectomatence", "quantgraphify", "dynasecize",
	"vibrmorphate", "plasmnetous", "xenotechity", "mechnomium",
	"cybermetrical", "lumiphonate", "aerologize", "peltogenable",
	"chronosystive", "syntechant", "neonomable", "cryptovalence",
	"polycorant", "megasphereic", "hypernetoid", "omnimatary",
	"exosecize", "tectgraphate", "quantmorphify", "dynastatous",
	"vibrmetrity", "plasmfluxium", "xenolithism", "mechnasian",
	"cybercorance", "luminetence", "aerosystant", "peltotechity",
	"chrononomium", "synvalism", "neocoroid", "cryptosphereary",
	"polymathate", "meganetize", "hypersecify", "omnigraphive",
	"exostatable", "tectmorphance", "quantfluxence", "dynalithant",
	"vibrmetrent", "plasmphonate", "xenologize", "mechnomify",
	"cybervalive", "lumicorable", "aerosystance", "peltotechent",
	"chrononomant", "synvality", "neocorium", "cryptosphereism",
	"polymathoid", "meganetary", "hypersecate", "omnigraphize",
	"exostatify", "tectmorphive", "quantfluxable", "dynalithance",
	"vibrmetrence", "plasmphonant", "xenologent", "mechnomtra",
	"cyberfluxian", "lumilithic", "aerometral", "peltophonous",
	"chronographity", "synstatium", "neologism", "polysphereary",
	"megatronate", "hypermatize", "omnisecify", "exocorive",
	"tectnetable", "quantvalance", "dynanomence", "vibrtechant",
	"plasmsysent", "xenonasal", "mechnomous", "cybermetrity",
	"lumiphonary", "aerographize", "peltostative", "chronologable",
	"synmorphance", "neogenent", "cryptosphereant", "polytronent",
	"megamatous", "hypersystic", "omnisecian", "exocoral",
	"tectnetous", "quantvality", "dynanomium", "vibrtechism",
	"plasmsysoid", "xenonastive", "mechnomable", "cyberfluxance",
	"lumilithical", "aerometrous", "peltophonium", "chronography",
	"synstatize", "neologify", "cryptogenive", "polyspherable",
	"megatronance", "hypermatence", "omnisecant", "exocorent",
	"tectnetant", "quantvalent", "dynanomity", "vibrtechium",
	"plasmsysism", "xenonasoid", "mechnomary", "cyberfluxate",
	"lumilithize", "aerometrify", "peltophonic", "chronographal",
	"synstatous", "neologity", "cryptogenium", "polysphereism",
	"megatronoid", "hypermatary", "omnisecate", "exocortra",
	"tectnetion", "quantvalian", "dynanomic", "vibrtechal",
	"plasmsysous", "xenonasity", "cyberfluxize", "lumilithify",
	"aerometrive", "peltophonable", "chronographance",
	"synstatence", "neologant", "cryptogenent", "polyspherentra",
	"megatronion", "hypermatian", "omnisecic",
}

// StealthWordGenerator 生成用于通配符校准的隐匿随机词。
type StealthWordGenerator struct {
	rng  *rand.Rand
	seen map[string]bool
}

// NewStealthWordGenerator 创建生成器。
func NewStealthWordGenerator() *StealthWordGenerator {
	return &StealthWordGenerator{rng: rand.New(rand.NewSource(rand.Int63())), seen: map[string]bool{}}
}

// Generate 生成一个未使用过的隐匿词（可排除指定词）。
func (g *StealthWordGenerator) Generate(omit map[string]bool) string {
	for i := 0; i < 1000; i++ {
		wordCount := 2 + g.rng.Intn(2)
		separator := stealthSeparators[g.rng.Intn(len(stealthSeparators))]
		candidate := g.candidateFromBank(wordCount, separator)

		if !g.isValid(candidate, omit) {
			continue
		}
		g.seen[candidate] = true
		return candidate
	}
	// 兜底：附加计数后缀保证唯一
	fallback := "loravian-" + strings.Repeat("x", 3) + strings.Repeat("y", 3)
	for i := 0; ; i++ {
		candidate := fallback + "-" + string(rune('a'+i%26)) + string(rune('a'+(i/26)%26))
		if !g.seen[candidate] && !omit[candidate] {
			g.seen[candidate] = true
			return candidate
		}
	}
}

// candidateFromBank 从词库采样组合候选词。
func (g *StealthWordGenerator) candidateFromBank(wordCount int, separator string) string {
	bank := stealthWordBank
	words := make([]string, 0, wordCount)
	used := map[int]bool{}
	for len(words) < wordCount {
		i := g.rng.Intn(len(bank))
		if used[i] {
			continue
		}
		used[i] = true
		words = append(words, bank[i])
	}
	return strings.Join(words, separator)
}

// isValid 校验候选词的合法性（长度、字符集、熵等）。
func (g *StealthWordGenerator) isValid(candidate string, omit map[string]bool) bool {
	if g.seen[candidate] || omit[candidate] {
		return false
	}
	if len(candidate) < 15 || len(candidate) > 30 {
		return false
	}
	if strings.ContainsAny(candidate[:1], "-_") || strings.ContainsAny(candidate[len(candidate)-1:], "-_") {
		return false
	}
	for _, bad := range []string{"--", "__", "-_", "_-"} {
		if strings.Contains(candidate, bad) {
			return false
		}
	}
	for _, c := range candidate {
		if !(c >= 'a' && c <= 'z' || c == '-' || c == '_') {
			return false
		}
	}
	if shannonEntropy(candidate) >= 3.95 {
		return false
	}

	normalized := strings.ReplaceAll(candidate, "_", "-")
	words := strings.Split(normalized, "-")
	if len(words) != 2 && len(words) != 3 {
		return false
	}
	for _, word := range words {
		if word == "" || commonDirectories[word] {
			return false
		}
	}
	return true
}

// shannonEntropy 计算字符串的香农熵。
func shannonEntropy(value string) float64 {
	counts := map[rune]int{}
	for _, c := range value {
		counts[c]++
	}
	length := float64(len(value))
	entropy := 0.0
	for _, count := range counts {
		p := float64(count) / length
		entropy -= p * math.Log2(p)
	}
	return entropy
}
