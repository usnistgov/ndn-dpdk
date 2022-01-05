import { Component, h, JSX } from "preact";

import type { BenchmarkOptions } from "./benchmark";

interface Props {
  opts: BenchmarkOptions;
  disabled: boolean;
  onChange: (update: Partial<BenchmarkOptions>) => void;
}

export class BenchmarkOptionsEditor extends Component<Props> {
  private id = `benchmark-options-form_${Math.random}_`;

  override render() {
    const {
      opts: {
        faceScheme,
        faceRxQueues,
        nFwds,
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
          <label for={`${this.id}faceScheme`}>face scheme</label>
          <select id={`${this.id}faceScheme`} value={faceScheme} disabled={disabled} onChange={this.handleUpdate("faceScheme", String as any)}>
            <option value="ether">Ethernet</option>
            <option value="vxlan">VXLAN</option>
          </select>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}faceRxQueues`}>face RX queues</label>
          <input id={`${this.id}faceRxQueues`} type="number" min="1" max="2" value={faceRxQueues} disabled={disabled} onChange={this.handleUpdate("faceRxQueues", Number)}/>
          <span class="pure-form-message-inline">Ethernet face can only use 1 RX queue</span>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}nFwds`}>forwarding threads</label>
          <input id={`${this.id}nFwds`} type="number" min="1" max="12" value={nFwds} disabled={disabled} onChange={this.handleUpdate("nFwds", Number)}/>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}payloadLen`}>payload length</label>
          <input id={`${this.id}payloadLen`} type="number" min="100" max="8000" step="100" value={payloadLen} disabled={disabled} onChange={this.handleUpdate("payloadLen", Number)}/>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}interestNameLen`}>Interest name length</label>
          <input id={`${this.id}interestNameLen`} type="number" min="3" max="15" value={interestNameLen} disabled={disabled} onChange={this.handleUpdate("interestNameLen", Number)}/>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}dataMatch`}>Data match</label>
          <select id={`${this.id}dataMatch`} value={dataMatch} disabled={disabled} onChange={this.handleUpdate("dataMatch", String as any)}>
            <option value="exact">exact match</option>
            <option value="prefix">prefix match</option>
          </select>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}duration`}>trial duration</label>
          <input id={`${this.id}duration`} type="number" min="10" max="300" value={duration} disabled={disabled} onChange={this.handleUpdate("duration", Number)}/>
          <span class="pure-form-message-inline">seconds</span>
        </div>
        {this.props.children}
      </fieldset>
    );
  }

  private handleUpdate<
    F extends keyof BenchmarkOptions,
    E extends Element & { value: string },
  >(field: F, parse: (s: string) => BenchmarkOptions[F]) {
    return (evt: JSX.TargetedEvent<E>) => {
      this.props.onChange({ [field]: parse(evt.currentTarget.value) });
    };
  }
}
