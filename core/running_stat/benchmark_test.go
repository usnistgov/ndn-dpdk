package running_stat

import (
	"testing"
)

func BenchmarkPush(b *testing.B) {
	s := New()
	benchmarkPush(s, b.N)
}
