package main

import (
	"sort"
	"time"

	"ndn-dpdk/app/ndnping"
)

type ThroughputBenchmarkConfig struct {
	IntervalMin  time.Duration // minimum TX interval to test for
	IntervalMax  time.Duration // maximum TX interval to test for
	IntervalStep time.Duration // TX interval step

	TxCount       int           // minimum number of Interests
	TxDurationMin time.Duration // minimum test duration
	TxDurationMax time.Duration // maximum test duration, unused if zero

	WarmupTime       time.Duration // ignore below-threshold satisfy ratio during warmup
	CooldownTime     time.Duration // wait period between stopping TX and stopping RX
	ReadCountersFreq time.Duration // how often to read counters

	SatisfyThreshold float64 // required Interest satisfy ratio
	RetestThreshold  float64 // retest after failure if Interest satisfy ratio is above this
	RetestCount      int     // maximum retests
}

// Benchmark throughput and find minimum sustained interval.
type ThroughputBenchmark struct {
	client *ndnping.Client
	cfg    ThroughputBenchmarkConfig

	intervals []time.Duration
	cnts      []ndnping.ClientCounters
}

func NewThroughputBenchmark(client *ndnping.Client, cfg ThroughputBenchmarkConfig) (tb *ThroughputBenchmark) {
	tb = new(ThroughputBenchmark)
	tb.client = client
	tb.cfg = cfg

	client.Stop()
	client.Tx.Stop()

	for interval := cfg.IntervalMin; interval <= cfg.IntervalMax; interval += cfg.IntervalStep {
		tb.intervals = append(tb.intervals, interval)
	}
	tb.cnts = make([]ndnping.ClientCounters, len(tb.intervals))

	return tb
}

// Run the benchmark to find minimum sustained interval.
func (tb *ThroughputBenchmark) Run() (ok bool, msi time.Duration, cnt ndnping.ClientCounters) {
	tblog.Infof("searching MSI within [%v,%v]", tb.intervals[0], tb.intervals[len(tb.intervals)-1])
	nTests := 0
	i := sort.Search(len(tb.intervals), func(i int) (ok bool) {
		interval := tb.intervals[i]
		for j := 0; j <= tb.cfg.RetestCount; j++ {
			nTests++
			ok, tb.cnts[i] = tb.Once(interval)
			dataRatio, _ := tb.cnts[i].ComputeRatios()
			if ok || dataRatio < tb.cfg.RetestThreshold {
				break
			}
		}
		return ok
	})
	if i == len(tb.intervals) {
		tblog.Infof("MSI out of range, after %d tests", nTests)
		return false, 0, ndnping.ClientCounters{}
	}
	tblog.Infof("MSI is %v, after %d tests", tb.intervals[i], nTests)
	return true, tb.intervals[i], tb.cnts[i]
}

// Test once with specified TX interval.
// Returns whether test passed and last counters.
func (tb *ThroughputBenchmark) Once(interval time.Duration) (ok bool, cnt ndnping.ClientCounters) {
	tb.client.ClearCounters()
	tb.client.SetInterval(interval)
	tb.client.Launch()
	tb.client.Tx.Launch()

	txDuration := interval * time.Duration(tb.cfg.TxCount)
	if txDuration < tb.cfg.TxDurationMin {
		txDuration = tb.cfg.TxDurationMin
	}
	if tb.cfg.TxDurationMax > 0 && txDuration > tb.cfg.TxDurationMax {
		txDuration = tb.cfg.TxDurationMax
	}
	tblog.Debugf("interval %dns, duration %0.2fs, satisfy threshold %0.2f%%",
		interval.Nanoseconds(), txDuration.Seconds(), tb.cfg.SatisfyThreshold*100)

	startTime := time.Now()
	stopTxTimer := time.NewTimer(txDuration)
	time.Sleep(tb.cfg.WarmupTime)
	readCountersTicker := time.NewTicker(tb.cfg.ReadCountersFreq)
	var stopRxTimer *time.Timer
	var stopRxTimerC <-chan time.Time = make(chan time.Time)

L:
	for {
		select {
		case now := <-readCountersTicker.C:
			cnt = tb.client.ReadCounters()
			dataRatio, _ := cnt.ComputeRatios()
			if dataRatio < tb.cfg.SatisfyThreshold {
				tblog.Debugf("early fail after %0.2fs", now.Sub(startTime).Seconds())
				break L
			}
		case <-stopTxTimer.C:
			tb.client.Tx.Stop()
			stopRxTimer = time.NewTimer(tb.cfg.CooldownTime)
			stopRxTimerC = stopRxTimer.C
		case <-stopRxTimerC:
			break L
		}
	}

	tb.client.Tx.Stop()
	tb.client.Stop()
	readCountersTicker.Stop()
	stopTxTimer.Stop()
	if stopRxTimer != nil {
		stopRxTimer.Stop()
	}

	cnt = tb.client.ReadCounters()
	dataRatio, _ := cnt.ComputeRatios()
	ok = dataRatio >= tb.cfg.SatisfyThreshold
	if ok {
		tblog.Debugf("satisfy ratio %0.2f%% above threshold", dataRatio*100)
	} else {
		tblog.Debugf("satisfy ratio %0.2f%% below threshold", dataRatio*100)
	}
	return
}
