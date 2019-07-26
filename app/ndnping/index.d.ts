import { Counter, NNDuration } from "../../core";

export as namespace ndnping;

interface ClientPacketCounters {
  NInterests: Counter;
  NData: Counter;
  NNacks: Counter;
}

interface ClientRttCounters {
  Min: NNDuration;
  Max: NNDuration;
  Avg: NNDuration;
  Stdev: NNDuration;
}

interface ClientPatternCounters extends ClientPacketCounters {
  Rtt: ClientRttCounters;
  NRttSamples: Counter;
}

export interface ClientCounters extends ClientPacketCounters {
  NAllocError: Counter;
  Rtt: ClientRttCounters;
  PerPattern: ClientPatternCounters[];
}
