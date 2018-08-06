import ndnjs = require("ndn-js");
import { DecodingException } from "ndn-js/js/encoding/decoding-exception.js";
import { TlvDecoder } from "ndn-js/js/encoding/tlv/tlv-decoder.js";
import { Tlv } from "ndn-js/js/encoding/tlv/tlv.js";

import { TT } from "./tlv-type";

export enum PktType {
  None = 0,
  Interest = TT.Interest,
  Data = TT.Data,
  Nack = TT.Nack,
}

const wf = ndnjs.TlvWireFormat.get();

export class Packet {
  public buf: Buffer;
  public netPkt: Buffer;
  public type: PktType = PktType.None;
  public pitToken: Uint8Array;

  public interest: ndnjs.Interest;
  public data: ndnjs.Data;
  public nack: ndnjs.NetworkNack;

  constructor(buf: Buffer) {
    this.buf = Buffer.from(buf);
    this.netPkt = this.buf;
    this.parsePacket(false);
  }

  public toString(): string {
    switch (this.type) {
      case PktType.Interest:
        return "I " + this.interest.getName().toUri();
      case PktType.Data:
        return "D " + this.data.getName().toUri();
      case PktType.Nack:
        return "N " + this.interest.getName().toUri() + "~" + this.nack.getReason();
    }
    return "X";
  }

  private parsePacket(isNested: boolean) {
    switch (this.netPkt[0]) {
      case TT.Interest:
        this.interest = new ndnjs.Interest();
        wf.decodeInterest(this.interest, this.netPkt);
        if (this.type !== PktType.Nack) {
          this.type = PktType.Interest;
        }
        break;
      case TT.Data:
        this.data = new ndnjs.Data();
        wf.decodeData(this.data, this.netPkt);
        this.type = PktType.Data;
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
          this.pitToken = tlvValue;
          break;
        case TT.Nack:
          this.nack = new ndnjs.NetworkNack();
          const code = d.readOptionalNonNegativeIntegerTlv(TT.NackReason, tlvValueEnd);
          if (code > 0) {
            this.nack.setReason(code);
            this.type = PktType.Nack;
          }
          break;
        case TT.LpPayload:
          this.netPkt = tlvValue;
          this.parsePacket(true);
          break;
        default:
          const canIgnore = tlvType >= Tlv.LpPacket_IGNORE_MIN &&
                          tlvType <= Tlv.LpPacket_IGNORE_MAX &&
                          (tlvType & 0x01) === 1;
          if (!canIgnore) {
            throw new DecodingException();
          }
          break;
      }
      d.seek(tlvValueEnd);
    }
  }
}
