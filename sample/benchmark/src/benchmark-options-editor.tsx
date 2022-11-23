import { type JSX, Component, h } from "preact";

import type { BenchmarkOptions } from "./benchmark";

interface Props {
  opts: BenchmarkOptions;
  disabled: boolean;
  onChange: (update: Partial<BenchmarkOptions>) => void;
}

function parseToString(s: string): any {
  return s;
}

function parseToInteger(s: string): any {
  return Number.parseInt(s, 10);
}

function alsoUpdateRxQueues(field: keyof BenchmarkOptions) {
  return (scheme: BenchmarkOptions.FaceScheme): Partial<BenchmarkOptions> | undefined => {
    if (scheme !== "vxlan") {
      return { [field]: 1 };
    }
    return undefined;
  };
}

function alsoUpdateDataMatch(producerKind: BenchmarkOptions.ProducerKind): Partial<BenchmarkOptions> | undefined {
  if (producerKind !== "pingserver") {
    return { dataMatch: "exact" };
  }
  return undefined;
}

export class BenchmarkOptionsEditor extends Component<Props> {
  private readonly id = globalThis.crypto.randomUUID();

  override render() {
    const {
      opts: {
        faceAScheme,
        faceARxQueues,
        faceBScheme,
        faceBRxQueues,
        nFwds,
        producerKind,
        interestNameLen,
        dataMatch,
        payloadLen,
        duration,
      },
      disabled,
    } = this.props;

    return (
      <fieldset class="benchmark-options-editor">
        <div class="pure-control-group">
          <label for={`${this.id}.faceAScheme`}>face A scheme</label>
          <select id={`${this.id}.faceAScheme`} value={faceAScheme} disabled={disabled} onChange={this.handleUpdate("faceAScheme", parseToString, alsoUpdateRxQueues("faceARxQueues"))}>
            <option value="ether">Ethernet</option>
            <option value="vxlan">VXLAN</option>
            <option value="memif">memif</option>
          </select>
        </div>
        <div class="pure-control-group" hidden={faceAScheme !== "vxlan"}>
          <label for={`${this.id}.faceARxQueues`}>face A RX queues</label>
          <input id={`${this.id}.faceARxQueues`} type="number" min="1" max="2" value={faceARxQueues} disabled={disabled} onChange={this.handleUpdate("faceARxQueues", parseToInteger)}/>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}.faceBScheme`}>face B scheme</label>
          <select id={`${this.id}.faceBScheme`} value={faceBScheme} disabled={disabled} onChange={this.handleUpdate("faceBScheme", parseToString, alsoUpdateRxQueues("faceBRxQueues"))}>
            <option value="ether">Ethernet</option>
            <option value="vxlan">VXLAN</option>
            <option value="memif">memif</option>
          </select>
        </div>
        <div class="pure-control-group" hidden={faceBScheme !== "vxlan"}>
          <label for={`${this.id}.faceBRxQueues`}>face B RX queues</label>
          <input id={`${this.id}.faceBRxQueues`} type="number" min="1" max="2" value={faceBRxQueues} disabled={disabled} onChange={this.handleUpdate("faceBRxQueues", parseToInteger)}/>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}.nFwds`}>forwarding threads</label>
          <input id={`${this.id}.nFwds`} type="number" min="1" max="12" value={nFwds} disabled={disabled} onChange={this.handleUpdate("nFwds", parseToInteger)}/>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}.producerKind`}>producer kind</label>
          <select id={`${this.id}.producerKind`} value={producerKind} disabled={disabled} onChange={this.handleUpdate("producerKind", parseToString, alsoUpdateDataMatch)}>
            <option value="pingserver">pingserver</option>
            <option value="fileserver">fileserver</option>
          </select>
          <span class="pure-form-message-inline">not implemented</span>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}.payloadLen`}>payload length</label>
          <input id={`${this.id}.payloadLen`} type="number" min="100" max="8000" step="100" value={payloadLen} disabled={disabled} onChange={this.handleUpdate("payloadLen", parseToInteger)}/>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}.interestNameLen`}>Interest name length</label>
          <input id={`${this.id}.interestNameLen`} type="number" min="3" max="15" value={interestNameLen} disabled={disabled} onChange={this.handleUpdate("interestNameLen", parseToInteger)}/>
        </div>
        <div class="pure-control-group" hidden={producerKind !== "pingserver"}>
          <label for={`${this.id}.dataMatch`}>Data match</label>
          <select id={`${this.id}.dataMatch`} value={dataMatch} disabled={disabled} onChange={this.handleUpdate("dataMatch", parseToString)}>
            <option value="exact">exact match</option>
            <option value="prefix">prefix match</option>
          </select>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}.duration`}>trial duration</label>
          <input id={`${this.id}.duration`} type="number" min="10" max="300" value={duration} disabled={disabled} onChange={this.handleUpdate("duration", parseToInteger)}/>
          <span class="pure-form-message-inline">seconds</span>
        </div>
        {this.props.children}
      </fieldset>
    );
  }

  private handleUpdate<
    F extends keyof BenchmarkOptions,
    E extends Element & { value: string },
  >(field: F, parse: (s: string) => BenchmarkOptions[F], also?: (value: BenchmarkOptions[F]) => Partial<BenchmarkOptions> | undefined) {
    return (evt: JSX.TargetedEvent<E>) => {
      const value = parse(evt.currentTarget.value);
      const update: Partial<BenchmarkOptions> = { [field]: value };
      Object.assign(update, also?.(value));
      this.props.onChange(update);
    };
  }
}
