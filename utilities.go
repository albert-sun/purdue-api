package purdue_api

import "time"

func reverseDates(s []time.Time) {
	// shamelessly copied from github
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}

	return 0
}

func abs(val int) int {
	if val >= 0 {
		return val
	}

	return -val
}

func stringArrContains(array []string, value string) bool {
	for _, val := range array {
		if val == value {
			return true
		}
	}

	return false
}
