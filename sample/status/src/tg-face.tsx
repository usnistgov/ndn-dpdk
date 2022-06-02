import type { FileServerMount, TgcPattern, TgpPattern } from "@usnistgov/ndn-dpdk";
import numd from "numd";
import { Component, Fragment, h } from "preact";

import { FileServerCounterView } from "./file-server-counter-view";
import { FileServerMountsTable } from "./file-server-mounts-table";
import { describeFaceLocator, Face as FaceB, Worker } from "./model";
import { TgcPatternsTable } from "./tgc-patterns-table";
import { TgpPatternsTable } from "./tgp-patterns-table";
import { WorkerShape } from "./worker-shape";

export interface Face extends FaceB {
  txLoop: Worker<"TX">;
  trafficgen: {
    id: string;
    rxLoops: Array<Worker<"RX">>;
    producer?: {
      patterns: TgpPattern[];
      workers: Worker[];
    };
    fileServer?: {
      mounts: FileServerMount[];
      workers: Worker[];
    };
    consumer?: {
      patterns: TgcPattern[];
      workers: Worker[];
    };
    fetcher?: {
      workers: Worker[];
    };
  };
}
export namespace Face {
  export const subselection = `
    ${FaceB.subselection}
    txLoop { ${Worker.subselection} }
    trafficgen {
      id
      rxLoops { ${Worker.subselection} }
      producer { patterns workers { ${Worker.subselection} } }
      fileServer { mounts workers { ${Worker.subselection} } }
      consumer { patterns workers { ${Worker.subselection} } }
      fetcher { workers { ${Worker.subselection} } }
    }
  `;
}

interface Props {
  face: Face;
}

export class TgFace extends Component<Props> {
  override render() {
    const { id: faceID, nid, locator, txLoop, trafficgen: { id, rxLoops, producer, fileServer, consumer, fetcher } } = this.props.face;
    const nElements = ((producer || fileServer ? 1 : 0) + (consumer || fetcher ? 1 : 0)) as 1 | 2;
    const [height, rxtxY, producerY, consumerY] = {
      1: [50, 0, 0, 0],
      2: [150, 50, 0, 100],
    }[nElements];
    return (
      <section>
        <h3 title={`face: ${faceID}\ntrafficgen: ${id}`}>
          {nid} <span title={JSON.stringify(locator, undefined, 2)}>{describeFaceLocator(locator)}</span>
        </h3>
        <svg style="background: #ffffff; width: 500px; font-size: 16px;" viewBox={`0 0 500 ${height}`}>
          {(producerY === consumerY ? [producerY] : [producerY, consumerY]).map((y) => (
            <>
              <line key={`${y}L`} x1={100} y1={rxtxY + 25} x2={150} y2={y + 25} stroke="#aaaaaa" stroke-width="1"/>
              <line key={`${y}R`} x1={400} y1={rxtxY + 25} x2={350} y2={y + 25} stroke="#aaaaaa" stroke-width="1"/>
            </>
          ))}
          <WorkerShape role="RX" label={`input ${gatherWorkerIDs(rxLoops)}`} x={0} y={rxtxY} width={100} height={50}/>
          <WorkerShape role="TX" label={`output ${gatherWorkerIDs([txLoop])}`} x={400} y={rxtxY} width={100} height={50}/>
          {producer && (
            <WorkerShape role="PRODUCER" label={`producer ${gatherWorkerIDs(producer.workers)}`} x={150} y={producerY} width={200} height={50}/>
          )}
          {fileServer && (
            <WorkerShape role="PRODUCER" label={`file server ${gatherWorkerIDs(fileServer.workers)}`} x={150} y={producerY} width={200} height={50}/>
          )}
          {consumer && (
            <WorkerShape role="CONSUMER" label={`consumer ${gatherWorkerIDs(consumer.workers)}`} x={150} y={consumerY} width={200} height={50}/>
          )}
          {fetcher && (
            <WorkerShape role="CONSUMER" label={`fetcher ${gatherWorkerIDs(fetcher.workers)}`} x={150} y={consumerY} width={200} height={50}/>
          )}
        </svg>
        {producer && (
          <details style="margin-top: 1em;">
            <summary>Producer, {numd(producer.patterns.length, "pattern", "patterns")}</summary>
            <TgpPatternsTable tgID={id} patterns={producer.patterns}/>
          </details>
        )}
        {fileServer && (
          <details style="margin-top: 1em;">
            <summary>File Server, {numd(fileServer.mounts.length, "mount entry", "mount entries")}</summary>
            <FileServerMountsTable mounts={fileServer.mounts}/>
            <FileServerCounterView tgID={id}/>
          </details>
        )}
        {consumer && (
          <details style="margin-top: 1em;">
            <summary>Consumer, {numd(consumer.patterns.length, "pattern", "patterns")}</summary>
            <TgcPatternsTable tgID={id} patterns={consumer.patterns}/>
          </details>
        )}
        {fetcher && (
          <details style="margin-top: 1em;">
            <summary>Fetcher</summary>
            <p>Congestion-aware fetcher is present on this face.</p>
          </details>
        )}
      </section>
    );
  }
}

function gatherWorkerIDs(workers: readonly Worker[]): string {
  return workers.map((w) => w.nid).join(", ");
}
