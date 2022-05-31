import type { TgcPattern, TgpPattern } from "@usnistgov/ndn-dpdk";
import { Component, Fragment, h } from "preact";

import { describeFaceLocator, Face as FaceB, Worker } from "./model";
import { TgConsumer } from "./tg-consumer";
import { TgProducer } from "./tg-producer";
import { WorkerShape } from "./worker-shape";

export interface Face extends FaceB {
  trafficgen: {
    id: string;
    producer?: {
      patterns: TgpPattern[];
      workers: Worker[];
    };
    fileServer?: {
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
    trafficgen {
      id
      producer { patterns workers { ${Worker.subselection} } }
      fileServer { workers { ${Worker.subselection} } }
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
    const { id: faceID, nid, locator, trafficgen: { id, producer, fileServer, consumer, fetcher } } = this.props.face;
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
        <svg style="background: #ffffff; width: 900px;" viewBox={`0 0 500 ${height}`}>
          <WorkerShape role="RX" label="input" x={0} y={rxtxY} width={100} height={50}/>
          <WorkerShape role="TX" label="output" x={400} y={rxtxY} width={100} height={50}/>
          {producer ? (
            <WorkerShape role="PRODUCER" label={`producer ${gatherWorkerIDs(producer)}`} x={150} y={producerY} width={200} height={50}/>
          ) : undefined}
          {fileServer ? (
            <WorkerShape role="PRODUCER" label={`file server ${gatherWorkerIDs(fileServer)}`} x={150} y={producerY} width={200} height={50}/>
          ) : undefined}
          {consumer ? (
            <WorkerShape role="CONSUMER" label={`consumer ${gatherWorkerIDs(consumer)}`} x={150} y={consumerY} width={200} height={50}/>
          ) : undefined}
          {fetcher ? (
            <WorkerShape role="CONSUMER" label={`fetcher ${gatherWorkerIDs(fetcher)}`} x={150} y={consumerY} width={200} height={50}/>
          ) : undefined}
          {(producerY === consumerY ? [producerY] : [producerY, consumerY]).map((y) => (
            <>
              <line key={`${y}L`} x1={100} y1={rxtxY + 25} x2={150} y2={y + 25} stroke="#111111" stroke-width="1"/>
              <line key={`${y}R`} x1={400} y1={rxtxY + 25} x2={350} y2={y + 25} stroke="#111111" stroke-width="1"/>
            </>
          ))}
        </svg>
        {producer && <TgProducer tgID={id} patterns={producer.patterns}/>}
        {consumer && <TgConsumer tgID={id} patterns={consumer.patterns}/>}
      </section>
    );
  }
}

function gatherWorkerIDs({ workers }: { workers: readonly Worker[] }): string {
  return workers.map((w) => w.nid).join(", ");
}
