import ndn = require("ndn-js");
import { DecodingException } from "ndn-js/js/encoding/decoding-exception.js";
import { TlvDecoder } from "ndn-js/js/encoding/tlv/tlv-decoder.js";
import { TlvEncoder } from "ndn-js/js/encoding/tlv/tlv-encoder.js";
import { Tlv } from "ndn-js/js/encoding/tlv/tlv.js";

import { TT } from "../../ndn/tlv-type";

export enum PktType {
  None = 0,
  Interest = TT.Interest,
  Data = TT.Data,
  Nack = TT.Nack,
}

export class Packet {
  public buf: Buffer;
  public netPkt: Buffer;
  public type: PktType = PktType.None;
  public pitToken?: string;

  public name?: ndn.Name;
  public interest?: ndn.Interest;
  public data?: ndn.Data;
  public nack?: ndn.NetworkNack;

  constructor(buf: Buffer) {
    this.buf = Buffer.from(buf);
    this.netPkt = this.buf;
    this.parsePacket(false);
  }

  public wireEncode(wantPitToken: boolean = true): Buffer {
    const e = new TlvEncoder();
    const len0 = e.getLength();
    e.writeBlobTlv(TT.LpPayload, this.netPkt);
    if (this.type === PktType.Nack) {
      const len1 = e.getLength();
      e.writeNonNegativeIntegerTlv(TT.NackReason, this.nack!.getReason());
      e.writeTypeAndLength(TT.Nack, e.getLength() - len1);
    }
    if (wantPitToken && this.pitToken) {
      e.writeBlobTlv(TT.PitToken, Buffer.from(this.pitToken, "hex"));
    }
    e.writeTypeAndLength(TT.LpPacket, e.getLength() - len0);
    return e.getOutput();
  }

  public toString(): string {
    switch (this.type) {
      case PktType.Interest:
        return "I " + this.interest!.getName().toUri();
      case PktType.Data:
        return "D " + this.data!.getName().toUri();
      case PktType.Nack:
        return "N " + this.interest!.getName().toUri() + "~" + this.nack!.getReason();
    }
    return "X";
  }

  private parsePacket(isNested: boolean) {
    switch (this.netPkt[0]) {
      case TT.Interest:
        this.interest = new ndn.Interest();
        this.interest.wireDecode(this.netPkt);
        if (this.type !== PktType.Nack) {
          this.type = PktType.Interest;
        }
        this.name = this.interest.getName();
        break;
      case TT.Data:
        this.data = new ndn.Data();
        this.data.wireDecode(this.netPkt);
        this.type = PktType.Data;
        this.name = this.data.getName();
        break;
      case TT.LpPacket:
        if (!isNested) {
          return this.parseLpPacket();
        }
        // LpPacket nested in LpPacket, fallthrough to failure
      default:
        throw new DecodingException();
    }
  }

  private parseLpPacket() {
    const d = new TlvDecoder(this.netPkt);
    const endOffset = d.readNestedTlvsStart(TT.LpPacket);
    while (d.getOffset() < endOffset) {
      const tlvType = d.readVarNumber();
      const tlvLength = d.readVarNumber();
      const tlvValueEnd = d.getOffset() + tlvLength;
      if (tlvValueEnd > endOffset) {
        throw new DecodingException();
      }
      const tlvValue = d.getSlice(d.getOffset(), tlvValueEnd);

      switch (tlvType) {
        case TT.PitToken:
          this.pitToken = tlvValue.toString("hex");
          break;
        case TT.Nack:
          this.nack = new ndn.NetworkNack();
          const code = d.readOptionalNonNegativeIntegerTlv(TT.NackReason, tlvValueEnd);
          if (code > 0) {
            (this.nack as any).setReason(code);
            this.type = PktType.Nack;
          }
          break;
        case TT.LpPayload:
          this.netPkt = tlvValue;
          this.parsePacket(true);
          break;
        default:
          // tslint:disable-next-line no-bitwise
          const lastBit = tlvType & 0x01;
          const canIgnore = tlvType >= Tlv.LpPacket_IGNORE_MIN &&
                            tlvType <= Tlv.LpPacket_IGNORE_MAX &&
                            lastBit === 1;
          if (!canIgnore) {
            throw new DecodingException();
          }
          break;
      }
      d.seek(tlvValueEnd);
    }
  }
}
