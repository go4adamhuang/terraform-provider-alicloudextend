package provider

import "strings"

func strPtr(s string) *string    { return &s }
func int64Ptr(i int64) *int64    { return &i }
func uint64Ptr(u uint64) *uint64 { return &u }

// splitImportID splits an import ID by ":" and returns the parts if exactly n parts are found.
func splitImportID(id string, n int) []string {
	parts := strings.SplitN(id, ":", n)
	if len(parts) != n {
		return nil
	}
	return parts
}
