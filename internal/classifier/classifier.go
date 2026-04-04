package classifier

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
)

const baseURL = "https://raw.githubusercontent.com/v2fly/domain-list-community/master/data/"

var serviceNames = map[string]string{
	"bilibili":      "B站",
	"youtube":       "YouTube",
	"google":        "Google",
	"baidu":         "百度",
	"qq":            "腾讯QQ",
	"wechat":        "微信",
	"tencent":       "腾讯",
	"taobao":        "淘宝",
	"alibaba":       "阿里巴巴",
	"alipay":        "支付宝",
	"jd":            "京东",
	"douyin":        "抖音",
	"tiktok":        "TikTok",
	"weibo":         "微博",
	"zhihu":         "知乎",
	"netease":       "网易",
	"163":           "网易163",
	"douban":        "豆瓣",
	"xiaomi":        "小米",
	"huawei":        "华为",
	"apple":         "Apple",
	"microsoft":     "Microsoft",
	"github":        "GitHub",
	"gitlab":        "GitLab",
	"stackoverflow": "StackOverflow",
	"openai":        "OpenAI",
	"anthropic":     "Anthropic",
	"cloudflare":    "Cloudflare",
	"amazon":        "Amazon",
	"aws":           "AWS",
	"netflix":       "Netflix",
	"spotify":       "Spotify",
	"twitter":       "Twitter/X",
	"facebook":      "Facebook",
	"instagram":     "Instagram",
	"whatsapp":      "WhatsApp",
	"telegram":      "Telegram",
	"discord":       "Discord",
	"reddit":        "Reddit",
	"wikipedia":     "Wikipedia",
	"steam":         "Steam",
	"epicgames":     "Epic Games",
	"nvidia":        "NVIDIA",
	"amd":           "AMD",
	"docker":        "Docker",
	"pypi":          "PyPI",
	"npm":           "NPM",
	"maven":         "Maven",
	"jetbrains":     "JetBrains",
	"vscode":        "VS Code",
	"zoom":          "Zoom",
	"slack":         "Slack",
	"notion":        "Notion",
	"feishu":        "飞书",
	"dingtalk":      "钉钉",
	"meituan":       "美团",
	"pinduoduo":     "拼多多",
	"xiaohongshu":   "小红书",
	"kuaishou":      "快手",
	"ctrip":         "携程",
	"didi":          "滴滴",
	"eleme":         "饿了么",
	"iqiyi":         "爱奇艺",
	"youku":         "优酷",
	"sohu":          "搜狐",
	"sina":          "新浪",
	"360":           "360",
	"kingsoft":      "金山",
	"adobe":         "Adobe",
	"dropbox":       "Dropbox",
	"mega":          "MEGA",
	"pornhub":       "PornHub",
	"xvideos":       "XVideos",
	"twitch":        "Twitch",
	"linkedin":      "LinkedIn",
	"bing":          "Bing",
	"duckduckgo":    "DuckDuckGo",
	"yahoo":         "Yahoo",
	"samsung":       "Samsung",
	"sony":          "Sony",
	"oracle":        "Oracle",
	"ibm":           "IBM",
	"hp":            "HP",
	"dell":          "Dell",
	"lenovo":        "联想",
}

// Classifier maps SNI domains to friendly service names using rules from
// v2fly/domain-list-community.
type Classifier struct {
	mu       sync.RWMutex
	suffixes map[string]string // domain suffix -> service name
	exacts   map[string]string // full domain -> service name
	logger   *slog.Logger
}

// New creates a new Classifier. Call Load to populate it with rules.
func New(logger *slog.Logger) *Classifier {
	return &Classifier{
		suffixes: make(map[string]string),
		exacts:   make(map[string]string),
		logger:   logger,
	}
}

// Load downloads and parses rules for all curated services.
// Call this once at startup. It's OK if some downloads fail.
func (c *Classifier) Load() {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // limit concurrent downloads

	for file, name := range serviceNames {
		wg.Add(1)
		go func(file, name string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			domains, err := c.downloadAndParse(file)
			if err != nil {
				if c.logger != nil {
					c.logger.Warn("failed to load domain rules", "service", file, "error", err)
				}
				return
			}

			c.mu.Lock()
			for _, d := range domains {
				if d.exact {
					c.exacts[d.domain] = name
				} else {
					c.suffixes[d.domain] = name
				}
			}
			c.mu.Unlock()
		}(file, name)
	}
	wg.Wait()

	c.mu.RLock()
	total := len(c.suffixes) + len(c.exacts)
	c.mu.RUnlock()

	if c.logger != nil {
		c.logger.Info("domain classifier loaded", "services", len(serviceNames), "rules", total)
	}
}

type domainRule struct {
	domain string
	exact  bool
}

func (c *Classifier) downloadAndParse(name string) ([]domainRule, error) {
	return c.fetchRules(name, 0)
}

func (c *Classifier) fetchRules(name string, depth int) ([]domainRule, error) {
	if depth > 5 {
		return nil, fmt.Errorf("include depth too deep")
	}

	resp, err := http.Get(baseURL + name)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var rules []domainRule
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Remove attributes like @ads @cn
		if idx := strings.Index(line, " @"); idx != -1 {
			line = line[:idx]
		}
		// Also remove trailing comments
		if idx := strings.Index(line, " #"); idx != -1 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "include:") {
			incName := strings.TrimPrefix(line, "include:")
			incRules, err := c.fetchRules(incName, depth+1)
			if err == nil {
				rules = append(rules, incRules...)
			}
			continue
		}

		if strings.HasPrefix(line, "regexp:") || strings.HasPrefix(line, "keyword:") {
			// Skip regex and keyword rules for simplicity
			continue
		}

		if strings.HasPrefix(line, "full:") {
			domain := strings.TrimPrefix(line, "full:")
			rules = append(rules, domainRule{domain: strings.ToLower(domain), exact: true})
		} else {
			// Suffix match
			domain := strings.ToLower(line)
			rules = append(rules, domainRule{domain: domain, exact: false})
		}
	}

	return rules, scanner.Err()
}

// Classify returns the friendly service name for a domain, or empty string if unknown.
func (c *Classifier) Classify(domain string) string {
	domain = strings.ToLower(domain)

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Try exact match first
	if name, ok := c.exacts[domain]; ok {
		return name
	}

	// Try suffix match: for "video.bilibili.com", try:
	//   "video.bilibili.com", "bilibili.com", "com"
	d := domain
	for {
		if name, ok := c.suffixes[d]; ok {
			return name
		}
		idx := strings.Index(d, ".")
		if idx == -1 {
			break
		}
		d = d[idx+1:]
	}

	return ""
}
