package string

import "testing"

func TestCapitalize(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "Test Capitalize",
			s:    "test",
			want: "Test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Capitalize(tt.s); got != tt.want {
				t.Errorf("Capitalize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultiTrim(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		prefixes []string
		suffixes []string
		want     string
	}{
		{
			name:     "Test MultiTrim",
			str:      "test",
			prefixes: []string{"t"},
			suffixes: []string{"t"},
			want:     "es",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MultiTrim(tt.str, tt.prefixes, tt.suffixes); got != tt.want {
				t.Errorf("MultiTrim() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRandStr(t *testing.T) {
	tests := []struct {
		name string
		len  int
	}{
		{
			name: "Test RandStr",
			len:  10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RandStr(tt.len); len(got) != tt.len {
				t.Errorf("RandStr() = %v, want %v", got, tt.len)
			}
		})
	}
}
