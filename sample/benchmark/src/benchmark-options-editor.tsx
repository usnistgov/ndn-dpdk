import assert from "minimalistic-assert";
import { type JSX, Component, h } from "preact";

import type { BenchmarkOptions } from "./benchmark";

interface Props {
  opts: BenchmarkOptions;
  disabled: boolean;
  onChange: (update: Partial<BenchmarkOptions>) => void;
}

function onFaceSchemeChange(field: keyof BenchmarkOptions) {
  return (scheme: BenchmarkOptions.FaceScheme): Partial<BenchmarkOptions> => {
    if (scheme !== "vxlan") {
      return { [field]: 1 };
    }
    return {};
  };
}

function onFwdsChange(nFwds: number, { nFwds: oldFwds, nFlows }: Readonly<BenchmarkOptions>): Partial<BenchmarkOptions> {
  return { nFlows: Math.ceil(nFlows / oldFwds) * nFwds };
}

function onProducerKindChange(producerKind: BenchmarkOptions.ProducerKind): Partial<BenchmarkOptions> {
  if (producerKind !== "pingserver") {
    return { interestNameLen: 5, dataMatch: "exact" };
  }
  return {};
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
        nFlows,
        trafficDir,
        producerKind,
        nProducerThreads,
        interestNameLen,
        dataMatch,
        payloadLen,
        segmentEnd,
        warmup,
        duration,
      },
      disabled,
    } = this.props;

    return (
      <fieldset class="benchmark-options-editor">
        <div class="pure-control-group">
          <label for={`${this.id}.faceAScheme`}>face A scheme</label>
          <select id={`${this.id}.faceAScheme`} value={faceAScheme} disabled={disabled} onChange={this.handleUpdate("faceAScheme", onFaceSchemeChange("faceARxQueues"))}>
            <option value="ether">Ethernet</option>
            <option value="vxlan">VXLAN</option>
            <option value="memif">memif</option>
          </select>
        </div>
        <div class="pure-control-group" hidden={faceAScheme !== "vxlan"}>
          <label for={`${this.id}.faceARxQueues`}>face A RX queues</label>
          <input id={`${this.id}.faceARxQueues`} type="number" min="1" max="2" value={faceARxQueues} disabled={disabled} onChange={this.handleUpdate("faceARxQueues")}/>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}.faceBScheme`}>face B scheme</label>
          <select id={`${this.id}.faceBScheme`} value={faceBScheme} disabled={disabled} onChange={this.handleUpdate("faceBScheme", onFaceSchemeChange("faceBRxQueues"))}>
            <option value="ether">Ethernet</option>
            <option value="vxlan">VXLAN</option>
            <option value="memif">memif</option>
          </select>
        </div>
        <div class="pure-control-group" hidden={faceBScheme !== "vxlan"}>
          <label for={`${this.id}.faceBRxQueues`}>face B RX queues</label>
          <input id={`${this.id}.faceBRxQueues`} type="number" min="1" max="2" value={faceBRxQueues} disabled={disabled} onChange={this.handleUpdate("faceBRxQueues")}/>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}.nFwds`}>forwarding threads</label>
          <input id={`${this.id}.nFwds`} type="number" min="1" max="12" value={nFwds} disabled={disabled} onChange={this.handleUpdate("nFwds", onFwdsChange)}/>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}.trafficDir`}>traffic direction</label>
          <select id={`${this.id}.trafficDir`} value={trafficDir} disabled={disabled} onChange={this.handleUpdate("trafficDir")}>
            <option value="2">bidirectional</option>
            <option value="1">unidirectional</option>
          </select>
          <span class="pure-form-message-inline" hidden={trafficDir !== 1}>producer A and consumer B</span>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}.producerKind`}>producer kind</label>
          <select id={`${this.id}.producerKind`} value={producerKind} disabled={disabled} onChange={this.handleUpdate("producerKind", onProducerKindChange)}>
            <option value="pingserver">pingserver</option>
            <option value="fileserver">fileserver</option>
          </select>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}.nProducerThreads`}>producer threads</label>
          <input id={`${this.id}.nProducerThreads`} type="number" min="1" max="2" value={nProducerThreads} disabled={disabled} onChange={this.handleUpdate("nProducerThreads")}/>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}.nFlows`}>fetcher flows</label>
          <input id={`${this.id}.nFlows`} type="number" min={nFwds} max="128" step={nFwds} value={nFlows} disabled={disabled} onChange={this.handleUpdate("nFlows")}/>
          <span class="pure-form-message-inline">
            {segmentEnd === 0 ? "" : `${Math.round(trafficDir * nFlows * payloadLen * segmentEnd / (1024 ** 3))} GB total`}
          </span>
        </div>
        <div class="pure-control-group" hidden={producerKind !== "pingserver"}>
          <label for={`${this.id}.interestNameLen`}>Interest name length</label>
          <input id={`${this.id}.interestNameLen`} type="number" min="4" max="15" value={interestNameLen} disabled={disabled} onChange={this.handleUpdate("interestNameLen")}/>
        </div>
        <div class="pure-control-group" hidden={producerKind !== "pingserver"}>
          <label for={`${this.id}.dataMatch`}>Data match</label>
          <select id={`${this.id}.dataMatch`} value={dataMatch} disabled={disabled} onChange={this.handleUpdate("dataMatch")}>
            <option value="exact">exact match</option>
            <option value="prefix">prefix match</option>
          </select>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}.payloadLen`}>payload length</label>
          <input id={`${this.id}.payloadLen`} type="number" min="100" max="8000" step="100" value={payloadLen} disabled={disabled} onChange={this.handleUpdate("payloadLen")}/>
          <span class="pure-form-message-inline">
            <input type="button" class="pure-button adjust-button" disabled={disabled} value="max" onClick={() => this.props.onChange({ payloadLen: 8000 })}/>
          </span>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}.segmentEnd`}>segment count</label>
          <input id={`${this.id}.segmentEnd`} type="number" min="0" max="1000000000000000" value={segmentEnd} disabled={disabled} onChange={this.handleUpdate("segmentEnd")}/>
          <span class="pure-form-message-inline">
            {segmentEnd === 0 ? "infinite" : `${Math.round(payloadLen * segmentEnd / (1024 ** 2))} MB`}
            {" "}
            <input type="button" class="pure-button adjust-button" disabled={disabled} value="-GB" onClick={() => this.adjustSegmentCount(-1)}/>
            {" "}
            <input type="button" class="pure-button adjust-button" disabled={disabled} value="+GB" onClick={() => this.adjustSegmentCount(+1)}/>
          </span>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}.warmup`}>warmup duration</label>
          <input id={`${this.id}.warmup`} type="number" min="0" max="30" step="5" value={warmup} disabled={disabled} onChange={this.handleUpdate("warmup")}/>
          <span class="pure-form-message-inline">seconds</span>
        </div>
        <div class="pure-control-group">
          <label for={`${this.id}.duration`}>trial duration</label>
          <input id={`${this.id}.duration`} type="number" min="10" max="300" step="5" value={duration} disabled={disabled} onChange={this.handleUpdate("duration")}/>
          <span class="pure-form-message-inline">seconds {segmentEnd > 0 ? "or retrieval completion" : ""}</span>
        </div>
        {this.props.children}
      </fieldset>
    );
  }

  private handleUpdate<
    F extends keyof BenchmarkOptions,
    E extends Element & { value: string },
  >(field: F, also?: (value: BenchmarkOptions[F], opts: Readonly<BenchmarkOptions>) => Partial<BenchmarkOptions>) {
    return (evt: JSX.TargetedEvent<E>) => {
      const { opts, onChange } = this.props;
      let value: any;
      switch (typeof opts[field]) {
        case "string":
          value = evt.currentTarget.value.trim();
          break;
        case "number":
          value = Number.parseInt(evt.currentTarget.value, 10);
          break;
        default:
          assert(false);
      }
      const update: Partial<BenchmarkOptions> = { [field]: value };
      Object.assign(update, also?.(value, opts));
      onChange(update);
    };
  }

  private adjustSegmentCount(change: number): void {
    const {
      opts: { payloadLen, segmentEnd },
      onChange,
    } = this.props;
    let gb = Math.round(payloadLen * segmentEnd / (1024 ** 3));
    gb += change;
    gb = Math.min(Math.max(0, gb), 32);
    onChange({ segmentEnd: Math.ceil(gb * (1024 ** 3) / payloadLen) });
  }
}
