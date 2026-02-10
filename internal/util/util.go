// Package util provides shared utility functions used across the project.
package util

// Contains returns true if the given slice contains the given item.
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
