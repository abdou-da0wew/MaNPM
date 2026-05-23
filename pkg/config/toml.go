package config

import (
	"fmt"
	"strconv"
	"strings"
)

func parseTOML(data []byte) (map[string]map[string]string, error) {
	sections := map[string]map[string]string{}
	lines := strings.Split(string(data), "\n")
	var currentSection string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") {
			end := strings.Index(line, "]")
			if end == -1 {
				return nil, fmt.Errorf("toml: unclosed section header: %s", line)
			}
			sectionName := strings.TrimSpace(line[1:end])
			if sectionName == "" {
				return nil, fmt.Errorf("toml: empty section name")
			}
			currentSection = sectionName
			if _, ok := sections[currentSection]; !ok {
				sections[currentSection] = map[string]string{}
			}
			continue
		}

		if currentSection == "" {
			continue
		}

		eqIdx := strings.Index(line, "=")
		if eqIdx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:eqIdx])
		rawValue := strings.TrimSpace(line[eqIdx+1:])

		rawValue = stripInlineComment(rawValue)

		if key == "" {
			continue
		}

		sections[currentSection][key] = rawValue
	}

	return sections, nil
}

func stripInlineComment(s string) string {
	inQuote := false
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			inQuote = !inQuote
			continue
		}
		if s[i] == '#' && !inQuote {
			return strings.TrimSpace(s[:i])
		}
	}
	return strings.TrimSpace(s)
}

func encodeTOML(sections map[string]map[string]string) ([]byte, error) {
	var b strings.Builder

	sectionOrder := make([]string, 0, len(sections))
	for s := range sections {
		sectionOrder = append(sectionOrder, s)
	}
	sortStrings(sectionOrder)

	for _, section := range sectionOrder {
		b.WriteString(fmt.Sprintf("[%s]\n", section))
		kv := sections[section]
		keys := make([]string, 0, len(kv))
		for k := range kv {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			v := kv[k]
			b.WriteString(fmt.Sprintf("%s = %s\n", k, v))
		}
		b.WriteString("\n")
	}

	return []byte(b.String()), nil
}

func sortStrings(s []string) {
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

func parseBool(s string) (bool, error) {
	switch s {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, fmt.Errorf("toml: invalid bool: %s", s)
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func parseString(s string) (string, error) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return "", fmt.Errorf("toml: invalid string: %s", s)
	}
	return s[1 : len(s)-1], nil
}

func isQuotedString(s string) bool {
	return len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"'
}

func isBool(s string) bool {
	return s == "true" || s == "false"
}

func isInt(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
