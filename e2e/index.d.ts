/* eslint-disable vars-on-top */
/* eslint-disable no-var */
import type { RpcClient } from './.env';
import type {
  CommandParameters,
} from './types';

declare global {
  var rpcClient: RpcClient<CommandParameters>;
  var latestRequestId: number;
  var latestPortId: number;
}

export {};
