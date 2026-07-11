package main

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsAddrInUse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "generic", err: errors.New("connection refused"), want: false},
		{
			name: "linux style",
			err:  errors.New("failed to listen: listen tcp4 0.0.0.0:8081: bind: address already in use"),
			want: true,
		},
		{
			name: "wrapped fiber",
			err:  fmt.Errorf("failed to listen: %w", errors.New("listen tcp4 0.0.0.0:8081: bind: address already in use")),
			want: true,
		},
		{
			name: "windows style",
			err:  errors.New("listen tcp 0.0.0.0:8081: bind: Only one usage of each socket address is normally permitted."),
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isAddrInUse(tc.err); got != tc.want {
				t.Fatalf("isAddrInUse(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
