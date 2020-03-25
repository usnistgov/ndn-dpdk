export interface Config {
  EnableEth?: boolean;
  EthDisableRxFlow?: boolean;
  EthMtu?: number;
  EthRxqFrames?: number;
  EthTxqPkts?: number;
  EthTxqFrames?: number;

  EnableSock?: boolean;
  SockTxqPkts?: number;
  SockTxqFrames?: number;

  EnableMock?: boolean;

  ChanRxgFrames?: number;
}
