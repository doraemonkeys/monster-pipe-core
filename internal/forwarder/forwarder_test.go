package forwarder

import "testing"

func Test_matchIPv4(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		ip      string
		want    bool
	}{
		{
			name:    "test1",
			pattern: "192.168.1.1",
			ip:      "192.168.1.1",
			want:    true,
		},
		{
			name:    "test2",
			pattern: "192.168.1.*",
			ip:      "192.168.1.1",
			want:    true,
		},
		{
			name:    "test3",
			pattern: "192.168.*",
			ip:      "192.168.1.1",
			want:    true,
		},
		{
			name:    "test4",
			pattern: "192.168.*.1",
			ip:      "192.168.61.1",
			want:    true,
		},
		{
			name:    "test5",
			pattern: "192.168.*.1",
			ip:      "192.168.61.2",
			want:    false,
		},
		{
			name:    "test6",
			pattern: "192.168.*.*",
			ip:      "192.168.61.1",
			want:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchIPv4(tt.pattern, tt.ip); got != tt.want {
				t.Errorf("matchIPv4() = %v, want %v", got, tt.want)
			}
		})
	}
}
