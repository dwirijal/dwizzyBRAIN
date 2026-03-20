import { deployNetwork } from './deploy.mjs';
import { defaultDeployNetworks } from './lib.mjs';

const networks = defaultDeployNetworks();

for (const network of networks) {
  console.log(`deploying ${network}...`);
  await deployNetwork(network);
}
