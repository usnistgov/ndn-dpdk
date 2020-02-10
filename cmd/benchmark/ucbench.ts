import Debug = require("debug");
import EventEmitter = require("events");
import StrictEventEmitter from "strict-event-emitter-types";

import { Uncertainty, UncertaintyState } from "./uncertainty.js";

const debug = Debug("ucbench");

interface Events<T> {
  progress: (observation: T, ucState: UncertaintyState) => void;
  oberror: (observation: T) => void;
  done: (ucState: UncertaintyState) => void;
}

type Emitter<T> = StrictEventEmitter<EventEmitter, Events<T>>;

export abstract class UcBenchmark<T> extends EventEmitter implements Emitter<T> {
  constructor(public readonly uncertainty: Uncertainty) {
    super();
    this.uncertainty = uncertainty;
  }

  public async run(): Promise<boolean> {
    let ucState = this.uncertainty.getState();
    while (!ucState.isSufficient) {
      const [res, value] = await this.observe();
      debug("observe result %j", res);
      if (isNaN(value)) {
        this.emit("oberror", res);
        return false;
      }

      this.uncertainty.addObservation(value);
      ucState = this.uncertainty.getState();
      debug("ucState %j", ucState);
      this.emit("progress", res, ucState);
    }
    this.emit("done", ucState);
    return true;
  }

  protected abstract observe(): Promise<[T, number]>;
}
