package purdue_api

// Utility functions I don't want to bloat the main files with.
func stringArrContains(array []string, value string) bool {
	for _, val := range array {
		if val == value {
			return true
		}
	}

	return false
} // checks whether an array contains a string
