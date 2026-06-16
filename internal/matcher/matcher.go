package matcher

import (
	"net"
	"regexp"
	"strconv"
	"strings"
)

var valueMatcherKeys = map[string]bool{
	"prefix":              true,
	"suffix":              true,
	"wildcard":            true,
	"equals-ignore-case":  true,
	"anything-but":        true,
	"exists":              true,
	"numeric":             true,
	"cidr":                true,
}

// Matches returns true if an event matches an EventBridge-style pattern.
// Both arguments should be unmarshalled JSON (map[string]interface{}).
func Matches(event, pattern map[string]interface{}) bool {
	for key, patternVal := range pattern {
		eventVal, exists := event[key]

		switch key {
		case "detail":
			if !matchDetail(eventVal, patternVal) {
				return false
			}
		case "resources":
			if !matchResources(eventVal, patternVal) {
				return false
			}
		default:
			if !exists {
				return false
			}
			if !matchField(eventVal, patternVal) {
				return false
			}
		}
	}
	return true
}

func matchField(eventVal interface{}, patternVal interface{}) bool {
	patterns, ok := asArray(patternVal)
	if !ok {
		return false
	}
	for _, m := range patterns {
		if matchValue(eventVal, m) {
			return true
		}
	}
	return false
}

func matchValue(eventVal interface{}, matcher interface{}) bool {
	switch m := matcher.(type) {
	case string:
		s, ok := eventVal.(string)
		return ok && s == m
	case float64:
		return matchExactNumeric(eventVal, m)
	case bool:
		b, ok := eventVal.(bool)
		return ok && b == m
	case nil:
		return eventVal == nil
	case map[string]interface{}:
		return matchObject(eventVal, m)
	case []interface{}:
		for _, sub := range m {
			if matchValue(eventVal, sub) {
				return true
			}
		}
		return false
	}
	return false
}

func matchExactNumeric(eventVal interface{}, n float64) bool {
	switch v := eventVal.(type) {
	case float64:
		return v == n
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return err == nil && f == n
	}
	return false
}

func matchObject(eventVal interface{}, matcher map[string]interface{}) bool {
	if existsVal, ok := matcher["exists"]; ok {
		b, _ := existsVal.(bool)
		return b == (eventVal != nil)
	}

	if eventVal == nil {
		return false
	}

	if prefix, ok := matcher["prefix"]; ok {
		s, _ := eventVal.(string)
		return strings.HasPrefix(s, prefix.(string))
	}
	if suffix, ok := matcher["suffix"]; ok {
		s, _ := eventVal.(string)
		return strings.HasSuffix(s, suffix.(string))
	}
	if eic, ok := matcher["equals-ignore-case"]; ok {
		s, _ := eventVal.(string)
		return strings.EqualFold(s, eic.(string))
	}
	if wc, ok := matcher["wildcard"]; ok {
		s, _ := eventVal.(string)
		return wildcardMatch(s, wc.(string))
	}
	if ab, ok := matcher["anything-but"]; ok {
		return anythingButMatch(eventVal, ab)
	}
	if numeric, ok := matcher["numeric"]; ok {
		return numericMatch(eventVal, numeric)
	}
	if cidr, ok := matcher["cidr"]; ok {
		s, _ := eventVal.(string)
		return cidrMatch(s, cidr.(string))
	}

	return false
}

func numericMatch(eventVal interface{}, numeric interface{}) bool {
	ops, ok := numeric.([]interface{})
	if !ok || len(ops) == 0 || len(ops)%2 != 0 {
		return false
	}

	var num float64
	switch v := eventVal.(type) {
	case float64:
		num = v
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return false
		}
		num = f
	default:
		return false
	}

	for i := 0; i < len(ops); i += 2 {
		op, _ := ops[i].(string)
		rhs, _ := toFloat(ops[i+1])
		switch op {
		case "<":
			if !(num < rhs) {
				return false
			}
		case "<=":
			if !(num <= rhs) {
				return false
			}
		case "=":
			if !(num == rhs) {
				return false
			}
		case ">=":
			if !(num >= rhs) {
				return false
			}
		case ">":
			if !(num > rhs) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	}
	return 0, false
}

func anythingButMatch(eventVal interface{}, matcher interface{}) bool {
	eventStr, ok := eventVal.(string)
	if !ok {
		return true
	}
	switch m := matcher.(type) {
	case []interface{}:
		for _, v := range m {
			if s, ok := v.(string); ok && eventStr == s {
				return false
			}
		}
		return true
	case map[string]interface{}:
		if prefix, ok := m["prefix"]; ok {
			return !strings.HasPrefix(eventStr, prefix.(string))
		}
		if suffix, ok := m["suffix"]; ok {
			return !strings.HasSuffix(eventStr, suffix.(string))
		}
	}
	return true
}

func cidrMatch(value, cidrPattern string) bool {
	_, n, err := net.ParseCIDR(cidrPattern)
	if err != nil {
		return false
	}
	ip := net.ParseIP(value)
	if ip == nil {
		return false
	}
	return n.Contains(ip)
}

func wildcardMatch(s, pattern string) bool {
	escaped := regexp.QuoteMeta(pattern)
	escaped = strings.ReplaceAll(escaped, `\*`, ".*")
	escaped = strings.ReplaceAll(escaped, `\?`, ".")
	re, err := regexp.Compile("^" + escaped + "$")
	if err != nil {
		return false
	}
	return re.MatchString(s)
}

func matchResources(eventVal interface{}, patternVal interface{}) bool {
	eventResources, ok := eventVal.([]interface{})
	if !ok {
		return false
	}
	patternArr, ok := asArray(patternVal)
	if !ok {
		return false
	}
	for _, pv := range patternArr {
		pvStr, ok := pv.(string)
		if !ok {
			continue
		}
		for _, ev := range eventResources {
			evStr, ok := ev.(string)
			if ok && evStr == pvStr {
				return true
			}
		}
	}
	return false
}

func matchDetail(eventVal interface{}, patternVal interface{}) bool {
	if eventVal == nil {
		return patternVal == nil
	}
	detailMap, ok := eventVal.(map[string]interface{})
	if !ok {
		return false
	}
	patternMap, ok := patternVal.(map[string]interface{})
	if !ok {
		return false
	}

	flatPattern := flattenDetail(patternMap, "")
	for path, patternFieldVal := range flatPattern {
		detailFieldVal := getNested(detailMap, path)
		if !matchField(detailFieldVal, patternFieldVal) {
			return false
		}
	}
	return true
}

func flattenDetail(pattern map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})
	for key, val := range pattern {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		if arr, ok := val.([]interface{}); ok {
			result[fullKey] = arr
		} else if nested, ok := val.(map[string]interface{}); ok {
			if isValueMatcher(nested) {
				result[fullKey] = []interface{}{nested}
			} else {
				for k, v := range flattenDetail(nested, fullKey) {
					result[k] = v
				}
			}
		} else {
			result[fullKey] = []interface{}{val}
		}
	}
	return result
}

func isValueMatcher(m map[string]interface{}) bool {
	for k := range m {
		if valueMatcherKeys[k] {
			return true
		}
	}
	return false
}

func getNested(obj map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = obj
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		val, exists := m[part]
		if !exists {
			return nil
		}
		current = val
	}
	return current
}

func asArray(v interface{}) ([]interface{}, bool) {
	switch a := v.(type) {
	case []interface{}:
		return a, true
	default:
		return []interface{}{v}, true
	}
}
