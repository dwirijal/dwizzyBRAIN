import 'dotenv/config';

import { ContractFactory, JsonRpcProvider, Wallet } from 'ethers';

import { loadSubscriptionManagerArtifact, readSecret, resolveNetworkConfig, writeDeploymentRecord } from './lib.mjs';

export async function deployNetwork(networkName) {
  const network = resolveNetworkConfig(networkName);
  const rpcUrl = process.env[network.rpcEnv]?.trim();
  if (!rpcUrl) {
    throw new Error(`${network.rpcEnv} is required`);
  }

  const privateKey = await readSecret('SUBSCRIPTION_DEPLOYER_PRIVATE_KEY');
  if (!privateKey) {
    throw new Error('SUBSCRIPTION_DEPLOYER_PRIVATE_KEY is required');
  }

  const ownerAddress = process.env.SUBSCRIPTION_OWNER_ADDRESS?.trim();
  const artifact = await loadSubscriptionManagerArtifact();
  const provider = new JsonRpcProvider(rpcUrl, network.chainId);
  const wallet = new Wallet(privateKey, provider);
  const resolvedOwner = ownerAddress || wallet.address;

  const currentNetwork = await provider.getNetwork();
  if (Number(currentNetwork.chainId) !== network.chainId) {
    throw new Error(`connected chain ${currentNetwork.chainId.toString()} does not match expected chain ${network.chainId}`);
  }

  const factory = new ContractFactory(artifact.abi, artifact.bytecode, wallet);
  const contract = await factory.deploy(resolvedOwner);
  await contract.waitForDeployment();

  const address = await contract.getAddress();
  const receipt = await contract.deploymentTransaction().wait();

  const record = {
    network: network.name,
    chainId: network.chainId,
    contractName: artifact.contractName,
    address,
    owner: resolvedOwner,
    deployer: wallet.address,
    txHash: receipt?.hash ?? contract.deploymentTransaction().hash ?? null,
    deployedAt: new Date().toISOString(),
  };

  const filePath = await writeDeploymentRecord(network.name, record);
  console.log(JSON.stringify({ ...record, filePath }, null, 2));
  return record;
}

async function main() {
  const networkName = process.argv[2] || process.env.SUBSCRIPTION_DEPLOY_NETWORK;
  if (!networkName) {
    throw new Error('network name is required');
  }
  await deployNetwork(networkName);
}

if (import.meta.url === `file://${process.argv[1]}`) {
  main().catch((err) => {
    console.error(err instanceof Error ? err.message : err);
    process.exit(1);
  });
}
