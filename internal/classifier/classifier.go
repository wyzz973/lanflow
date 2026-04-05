package classifier

import (
	"bufio"
	_ "embed"
	"log/slog"
	"strings"
)

//go:embed rules.txt
var rulesData string

//go:embed ndpi_rules.txt
var ndpiRulesData string

type Classifier struct {
	// v2fly rules (broader)
	suffixes map[string]string // domain suffix → service name
	exacts   map[string]string // full:domain → service name
	// nDPI rules (more specific app names, higher priority)
	ndpiSuffixes map[string]string
	ndpiExacts   map[string]string
	logger       *slog.Logger
}

func New(logger *slog.Logger) *Classifier {
	c := &Classifier{
		suffixes:     make(map[string]string),
		exacts:       make(map[string]string),
		ndpiSuffixes: make(map[string]string),
		ndpiExacts:   make(map[string]string),
		logger:       logger,
	}
	c.loadV2flyRules()
	c.loadNdpiRules()
	return c
}

// isGenericService returns true for meta-category services whose domains
// should not override a more specific service classification.
func isGenericService(fileKey string) bool {
	return strings.HasPrefix(fileKey, "category-") ||
		strings.HasPrefix(fileKey, "geolocation-") ||
		fileKey == "tld-cn" || fileKey == "tld-not-cn" ||
		fileKey == "cn"
}

func (c *Classifier) loadV2flyRules() {
	// Track the source file key for each domain so specific services
	// take priority over generic category/geolocation aggregates.
	suffixSource := make(map[string]string) // domain → file key
	exactSource := make(map[string]string)

	var currentService string
	var currentFileKey string
	scanner := bufio.NewScanner(strings.NewReader(rulesData))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			// Service header: "# bilibili:B站"
			header := line[2:]
			if idx := strings.Index(header, ":"); idx != -1 {
				currentFileKey = header[:idx]
				currentService = header[idx+1:]
			}
			continue
		}
		if currentService == "" {
			continue
		}

		if strings.HasPrefix(line, "full:") {
			domain := strings.ToLower(strings.TrimPrefix(line, "full:"))
			prev, exists := exactSource[domain]
			if !exists || (isGenericService(prev) && !isGenericService(currentFileKey)) {
				c.exacts[domain] = currentService
				exactSource[domain] = currentFileKey
			}
		} else {
			domain := strings.ToLower(line)
			prev, exists := suffixSource[domain]
			if !exists || (isGenericService(prev) && !isGenericService(currentFileKey)) {
				c.suffixes[domain] = currentService
				suffixSource[domain] = currentFileKey
			}
		}
	}

	if c.logger != nil {
		c.logger.Info("domain classifier loaded", "rules", len(c.suffixes)+len(c.exacts))
	}
}

func (c *Classifier) loadNdpiRules() {
	scanner := bufio.NewScanner(strings.NewReader(ndpiRulesData))
	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		domain := strings.ToLower(parts[0])
		appName := parts[1]

		// Make app names more human-friendly
		appName = humanizeAppName(appName)

		// If domain ends with ".", it's a prefix match in nDPI
		// We treat it as suffix match (same effect)
		if strings.HasSuffix(domain, ".") {
			domain = strings.TrimSuffix(domain, ".")
			c.ndpiSuffixes[domain] = appName
		} else if strings.Contains(domain, ".") {
			// Has a dot = could be exact or suffix
			// nDPI treats most as suffix matches
			c.ndpiSuffixes[domain] = appName
		} else {
			c.ndpiSuffixes[domain] = appName
		}
		count++
	}
	if c.logger != nil {
		c.logger.Info("nDPI rules loaded", "rules", count)
	}
}

func humanizeAppName(name string) string {
	// Convert CamelCase to spaces: "GoogleDrive" -> "Google Drive"
	var result []byte
	for i, ch := range name {
		if i > 0 && ch >= 'A' && ch <= 'Z' {
			prev := name[i-1]
			if prev >= 'a' && prev <= 'z' {
				result = append(result, ' ')
			}
		}
		result = append(result, byte(ch))
	}
	s := string(result)

	// Special cases
	replacer := strings.NewReplacer(
		"You Tube", "YouTube",
		"Git Lab", "GitLab",
		"Git Hub", "GitHub",
		"Linked In", "LinkedIn",
		"Tik Tok", "TikTok",
		"Drop Box", "Dropbox",
		"Click House", "ClickHouse",
		"Cloud Flare", "Cloudflare",
		"Bit Torrent", "BitTorrent",
		"Hot Spot", "HotSpot",
		"Game Pass", "GamePass",
		"i Cloud", "iCloud",
		"i QIYI", "iQIYI",
		"e Bay", "eBay",
		"YouTube Kids", "YouTube Kids",
		"YouTube Upload", "YouTube Upload",
		"Fb ook", "Facebook",
		"Fbook Reel Story", "Facebook Reel/Story",
		"Face Book", "Facebook",
		"Wh Ats App", "WhatsApp",
		"Co D_Mobile", "CoD Mobile",
		"Do H_Do T", "DoH/DoT",
		"Ge Force Now", "GeForce Now",
		"Gear UP_Booster", "GearUP Booster",
		"Lago Fast", "LagoFast",
		"Net Flix", "Netflix",
		"i Heart Radio", "iHeart Radio",
		"Di Rec TV", "DirecTV",
		"Sina Weibo", "新浪微博",
		"Ding Talk", "钉钉",
		"We Chat", "微信",
		"Net Ease Games", "网易游戏",
		"Tencent video", "腾讯视频",
		"Windows Update", "Windows Update",
		"Apple Push", "Apple Push",
		"Apple Siri", "Apple Siri",
		"Apple Store", "Apple Store",
		"Applei Cloud", "iCloud",
		"Applei Tunes", "iTunes",
		"Apple TVPlus", "Apple TV+",
		"Disney Plus", "Disney+",
		"Paramount Plus", "Paramount+",
		"Facebook Messenger", "Facebook Messenger",
		"Kakao Talk", "KakaoTalk",
		"Google Classroom", "Google Classroom",
		"Google Services", "Google Services",
		"Huawei Cloud", "华为云",
		"Private Internet Access", "Private Internet Access",
		"Proton VPN", "ProtonVPN",
		"Nord VPN", "NordVPN",
		"Surf Shark", "Surfshark",
		"Opera VPN", "OperaVPN",
		"Tunnel Bear", "TunnelBear",
		"World Of Warcraft", "World of Warcraft",
		"Path of Exile", "Path of Exile",
		"Riot Games", "Riot Games",
		"Epic Games", "Epic Games",
		"Electronic Arts", "EA Games",
		"Rockstar Games", "Rockstar Games",
		"Gaijin Entertainment", "Gaijin Entertainment",
		"Tesla Services", "Tesla Services",
		"Microsoft365", "Microsoft 365",
		"Yandex Alice", "Yandex Alice",
		"Yandex Cloud", "Yandex Cloud",
		"Yandex Direct", "Yandex Direct",
		"Yandex Disk", "Yandex Disk",
		"Yandex Mail", "Yandex Mail",
		"Yandex Market", "Yandex Market",
		"Yandex Metrika", "Yandex Metrika",
		"Yandex Music", "Yandex Music",
		"Sirius XMRadio", "SiriusXM Radio",
		"A WSKinesis", "AWS Kinesis",
		"AWS_Kinesis", "AWS Kinesis",
	)
	s = replacer.Replace(s)
	return s
}

// Classify returns the friendly service name for a domain, or empty string if unknown.
func (c *Classifier) Classify(domain string) string {
	domain = strings.ToLower(domain)

	// Priority 1: nDPI exact match
	if name, ok := c.ndpiExacts[domain]; ok {
		return name
	}

	// Priority 2: nDPI suffix match
	d := domain
	for {
		if name, ok := c.ndpiSuffixes[d]; ok {
			return name
		}
		idx := strings.Index(d, ".")
		if idx == -1 {
			break
		}
		d = d[idx+1:]
	}

	// Priority 3: v2fly exact match
	if name, ok := c.exacts[domain]; ok {
		return name
	}

	// Priority 4: v2fly suffix match
	d = domain
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
