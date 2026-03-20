import { compileSubscriptionManager } from './lib.mjs';

const artifact = await compileSubscriptionManager({ writeArtifact: true });
console.log(JSON.stringify({
  contractName: artifact.contractName,
  compiler: artifact.compiler,
  artifact: 'artifacts/SubscriptionManager.json',
}, null, 2));
