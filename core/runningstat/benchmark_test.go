package runningstat

import (
	"testing"
)

func BenchmarkPush(b *testing.B) {
	s := New()
	benchmarkPush(s, b.N)
}
