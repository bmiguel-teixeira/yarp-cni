package utils

func TruncateString(s string, max int) string {
	return s[:max]
}

func Contains(list []string, str string) bool {
	for _, b := range list {
		if b == str {
			return true
		}
	}
	return false
}

func Remove(s []string, str string) []string {
	for i, v := range s {
		if v == str {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}
