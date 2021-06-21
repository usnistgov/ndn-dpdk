package runningstattest

/*
#include "../../../csrc/core/running-stat.h"

void
RunningStat_BenchmarkPush(RunningStat* s, int n, bool enableMinMax)
{
	RunningStat_Clear(s, enableMinMax);
	if (enableMinMax) {
		for (int i = 0; i < n; ++i) {
			RunningStat_Push(s, i);
		}
	} else {
		for (int i = 0; i < n; ++i) {
			RunningStat_Push1(s, i);
		}
	}
}
*/
import "C"
import (
	"testing"
)

// Push n inputs.
func cbenchmarkPushMinMax(b *testing.B) {
	var s C.RunningStat
	C.RunningStat_BenchmarkPush(&s, C.int(b.N), true)
}

// Push n inputs.
func cbenchmarkPushNoMinMax(b *testing.B) {
	var s C.RunningStat
	C.RunningStat_BenchmarkPush(&s, C.int(b.N), false)
}
