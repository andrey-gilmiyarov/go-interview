package solution

import "strings"

func Join(parts []string) string {
	if len(parts) == 0 {
		return ""
	}

	size := len(parts) - 1
	for _, part := range parts {
		size += len(part)
	}

	var result strings.Builder
	result.Grow(size)
	for index, part := range parts {
		if index > 0 {
			result.WriteByte(',')
		}
		result.WriteString(part)
	}
	return result.String()
}
