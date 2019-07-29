import computeStdev = require("compute-stdev");
import * as _ from "lodash";

export interface UncertaintyState {
  /**
   * mean of observations
   */
  mean: number;

  /**
   * standard deviation of observations
   */
  stdev: number;

  /**
   * number of existing observations
   */
  count: number;

  /**
   * number of required observations
   */
  n: number;

  /**
   * whether sufficient number of observations have been completed
   */
  isSufficient: boolean;
}

/**
 * minimum number of observations for observed stdev to approach true stdev
 */
const MIN_N = 6;

/**
 * determine whether there are sufficient observations to achieve desired uncertainty with 95% confidence
 */
export class Uncertainty {
  private desiredUncertainty: number;
  private observations: number[];

  public constructor(desiredUncertainty: number) {
    this.desiredUncertainty = desiredUncertainty;
    this.observations = [];
  }

  public addObservation(value: number): void {
    this.observations.push(value);
  }

  public getState(): UncertaintyState {
    const stdev = computeStdev(this.observations);
    const count = this.observations.length;
    const n = Math.max(MIN_N, Math.ceil(Math.pow(2 * stdev / this.desiredUncertainty, 2)));
    return {
      mean: _.mean(this.observations),
      stdev,
      count,
      n,
      isSufficient: count >= n,
    };
  }
}
