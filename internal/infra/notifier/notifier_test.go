package notifier

import (
	"reflect"
	"testing"
)

func TestSplitRecipients(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"  ", nil},
		{"-1001", []string{"-1001"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{"a\nb\r\nc", []string{"a", "b", "c"}},
		{"https://x.com/a, https://y.com/b", []string{"https://x.com/a", "https://y.com/b"}},
	}
	for _, tt := range tests {
		got := SplitRecipients(tt.in)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("SplitRecipients(%q) = %#v; want %#v", tt.in, got, tt.want)
		}
	}
}
