package importer

import (
	"os"
	"reflect"
	"sort"
	"testing"
)

func TestLoadFilterList(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []int
	}{
		{
			name: "Standard Header",
			content: `Radio ID,Callsign,Name
1234567,N0XXX,Bob
7654321,N0YYY,Alice
`,
			expected: []int{1234567, 7654321},
		},
		{
			name: "Different Header Name",
			content: `foo,id,bar
1,1111111,x
2,2222222,y
`,
			expected: []int{1111111, 2222222},
		},
		{
			name: "Just IDs",
			content: `1234567
7654321
`,
			expected: []int{1234567, 7654321},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "filter_test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			ids, err := LoadFilterList(tmpfile.Name())
			if err != nil {
				t.Fatalf("LoadFilterList failed: %v", err)
			}

			// Sort for comparison
			keys := make([]int, 0, len(ids))
			for k := range ids {
				keys = append(keys, k)
			}
			sort.Ints(keys)
			sort.Ints(tt.expected)

			if !reflect.DeepEqual(keys, tt.expected) {
				t.Errorf("got %v, want %v", keys, tt.expected)
			}
		})
	}
}
