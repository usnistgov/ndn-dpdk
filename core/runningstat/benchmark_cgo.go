package runningstat

/*
#include "../../csrc/core/running-stat.h"

void
RunningStat_BenchmarkPush(RunningStat* s, int n)
{
	for (int i = 0; i < n; ++i) {
		RunningStat_Push(s, i);
	}
}
*/
import "C"

// Push n inputs.
func benchmarkPush(s *RunningStat, n int) {
	C.RunningStat_BenchmarkPush(s.v.getPtr(), C.int(n))
}
