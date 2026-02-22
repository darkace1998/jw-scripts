package util

import "testing"

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		item  string
		want  bool
	}{
		{"found", []string{"a", "b", "c"}, "b", true},
		{"not found", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
		{"nil slice", nil, "a", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Contains(tc.slice, tc.item); got != tc.want {
				t.Errorf("Contains(%v, %q) = %v, want %v", tc.slice, tc.item, got, tc.want)
			}
		})
	}
}
