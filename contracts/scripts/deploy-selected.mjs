import 'dotenv/config';

import { deployNetwork } from './deploy.mjs';
import { defaultDeployNetworks } from './lib.mjs';

function parseNetworks(value) {
  const raw = String(value || '').trim();
  if (raw === '') {
    return defaultDeployNetworks();
  }
  return raw
    .split(',')
    .map((entry) => entry.trim())
    .filter(Boolean);
}

const networks = parseNetworks(process.env.SUBSCRIPTION_DEPLOY_NETWORKS || process.argv.slice(2).join(','));

for (const network of networks) {
  console.log(`deploying ${network}...`);
  await deployNetwork(network);
}
