import fail = require("@zingle/fail");
import Debug = require("debug");
import delay = require("delay");
import * as _ from "lodash";
import moment = require("moment");
import * as yargs from "yargs";

import { NNDuration } from "../../core";

import { ITrafficGen, NdnpingTrafficGen, TrafficGenCounters } from "./trafficgen";

const debug = Debug("msi");

export interface Options {
  IntervalMin: NNDuration; /// minimum TX interval to test for
  IntervalMax: NNDuration; /// maximum TX interval to test for
  IntervalStep: NNDuration; /// TX interval step

  TxCount: number; // expected number of Interests
  TxDurationMin: number; /// minimum test duration (secs)
  TxDurationMax: number; /// maximum test duration (secs)

  WarmupTime: number; /// don't early fail during this warmup period (secs)
  CooldownTime: number; /// wait period (secs) between stopping TX and stopping RX
  ReadCountersFreq: number; /// how often (secs) to read counters

  SatisfyThreshold: number; /// pass if Interest satisfy ratio above
  EarlyFailThreshold: number; /// early-fail if Interest satisfy ratio below
}

async function runOnce(gen: ITrafficGen, interval: NNDuration, opt: Options): Promise<[boolean, TrafficGenCounters]> {
  await gen.start(interval);

  const txDuration = moment.duration(_.clamp(interval * opt.TxCount / 1e9,
                                             opt.TxDurationMin, opt.TxDurationMax), "s");
  const endTime = moment().add(txDuration);
  debug("interval=%d txDuration=%d ending-at=%s", interval, txDuration.asSeconds(), endTime.format());

  await delay(opt.WarmupTime * 1000);

  let lastSatisfyRatio: number = 0.0;
  let cnt: TrafficGenCounters;
  while (moment().isBefore(endTime)) {
    cnt = await gen.readCounters();
    if (cnt.satisfyRatio < Math.min(opt.EarlyFailThreshold, lastSatisfyRatio)) {
      debug("interval=%d early-fail satisfy-ratio=%d", interval, cnt.satisfyRatio);
      await gen.stop(moment.duration(0));
      return [false, cnt];
    }
    lastSatisfyRatio = cnt.satisfyRatio;
    await delay(opt.ReadCountersFreq * 1000);
  }

  await gen.stop(moment.duration(opt.CooldownTime, "s"));
  cnt = await gen.readCounters();
  const pass = cnt.satisfyRatio >= opt.SatisfyThreshold;
  debug("interval=%d %s satisfy-ratio=%d", interval, pass ? "pass" : "fail", cnt.satisfyRatio);
  return [pass, cnt];
}

interface MeasureResult {
  isUnderflow: boolean;
  isOverflow: boolean;
  MSI?: NNDuration;
  cnt?: TrafficGenCounters;
}

export async function measure(gen: ITrafficGen, options: Partial<Options> = {}): Promise<MeasureResult> {
  const opt: Options = Object.assign({
    IntervalMin: 500,
    IntervalMax: 3500,
    IntervalStep: 1,
    TxCount: 24000000,
    TxDurationMin: 15,
    TxDurationMax: 60,
    WarmupTime: 5,
    CooldownTime: 2,
    ReadCountersFreq: 1,
    SatisfyThreshold: 0.999,
    EarlyFailThreshold: 0.970,
  } as Options, options);

  const res: MeasureResult = {
    isUnderflow: true,
    isOverflow: true,
  };
  if (opt.IntervalMin > opt.IntervalMax) {
    return res;
  }

  const range = _.range(opt.IntervalMin, opt.IntervalMax + 1, opt.IntervalStep);
  let left = 0;
  let right = range.length - 1;
  while (left <= right) {
    const mid = left + Math.floor((right - left) / 2);
    const interval = range[mid];
    debug("range=[%d...%d...%d] rem-runs=%d", range[left], interval, range[right],
          Math.ceil(Math.log(right - left + 1) / Math.log(2)));
    const [pass, cnt] = await runOnce(gen, interval, opt);
    if (pass) {
      right = mid - 1;
      res.MSI = interval;
      res.cnt = cnt;
    } else {
      left = mid + 1;
    }
  }

  res.isUnderflow = right < 0;
  res.isOverflow = left >= range.length;
  return res;
}

async function main() {
  const argv = yargs.parse() as Partial<Options>;
  const gen = await NdnpingTrafficGen.create();
  const res = await measure(gen, argv);
  process.stdout.write(JSON.stringify(res) + "\n");
}

if (require.main === module) {
  main()
  .catch(fail);
}
