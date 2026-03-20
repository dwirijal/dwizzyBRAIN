import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import solc from 'solc';

export const rootDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
export const sourcePath = path.join(rootDir, 'src', 'SubscriptionManager.sol');
export const artifactDir = path.join(rootDir, 'artifacts');
export const artifactPath = path.join(artifactDir, 'SubscriptionManager.json');
export const deploymentsDir = path.join(rootDir, 'deployments');

export function resolveNetworkConfig(name) {
  const normalized = String(name || '').trim().toLowerCase();
  switch (normalized) {
    case 'base':
      return {
        name: 'base',
        chainId: 8453,
        rpcEnv: 'SUBSCRIPTION_BASE_RPC_URL',
      };
    case 'bsc':
    case 'bnb':
      return {
        name: 'bsc',
        chainId: 56,
        rpcEnv: 'SUBSCRIPTION_BSC_RPC_URL',
      };
    case 'hyperevm':
    case 'hyperliquid':
      return {
        name: 'hyperevm',
        chainId: 999,
        rpcEnv: 'SUBSCRIPTION_HYPEREVM_RPC_URL',
      };
    case 'kaia':
      return {
        name: 'kaia',
        chainId: 8217,
        rpcEnv: 'SUBSCRIPTION_KAIA_RPC_URL',
      };
    case 'sonic':
      return {
        name: 'sonic',
        chainId: 146,
        rpcEnv: 'SUBSCRIPTION_SONIC_RPC_URL',
      };
    case 'arbitrum':
      return {
        name: 'arbitrum',
        chainId: 42161,
        rpcEnv: 'SUBSCRIPTION_ARBITRUM_RPC_URL',
      };
    case 'abstract':
      return {
        name: 'abstract',
        chainId: 2741,
        rpcEnv: 'SUBSCRIPTION_ABSTRACT_RPC_URL',
      };
    default:
      throw new Error(`unsupported network ${name}`);
  }
}

export function defaultDeployNetworks() {
  return ['base', 'bsc', 'kaia', 'arbitrum'];
}

export async function readSecret(name) {
  const direct = String(process.env[name] || '').trim();
  if (direct !== '') {
    return direct;
  }
  const fileName = String(process.env[`${name}_FILE`] || '').trim();
  if (fileName === '') {
    return '';
  }
  const raw = await fs.readFile(fileName, 'utf8');
  return raw.trim();
}

export async function compileSubscriptionManager({ writeArtifact = true } = {}) {
  const source = await fs.readFile(sourcePath, 'utf8');
  const input = {
    language: 'Solidity',
    sources: {
      'SubscriptionManager.sol': { content: source },
    },
    settings: {
      optimizer: { enabled: true, runs: 200 },
      outputSelection: {
        '*': {
          '*': ['abi', 'evm.bytecode.object', 'evm.deployedBytecode.object'],
        },
      },
    },
  };

  const output = JSON.parse(solc.compile(JSON.stringify(input)));
  const errors = Array.isArray(output.errors) ? output.errors : [];
  const fatalErrors = errors.filter((entry) => entry.severity === 'error');
  if (fatalErrors.length > 0) {
    throw new Error(fatalErrors.map((entry) => entry.formattedMessage).join('\n'));
  }

  const contract = output.contracts?.['SubscriptionManager.sol']?.SubscriptionManager;
  if (!contract) {
    throw new Error('SubscriptionManager artifact was not produced');
  }

  const artifact = {
    contractName: 'SubscriptionManager',
    abi: contract.abi,
    bytecode: `0x${contract.evm.bytecode.object}`,
    deployedBytecode: `0x${contract.evm.deployedBytecode.object}`,
    compiler: solc.version(),
    sourcePath: 'src/SubscriptionManager.sol',
    generatedAt: new Date().toISOString(),
  };

  if (writeArtifact) {
    await fs.mkdir(artifactDir, { recursive: true });
    await fs.writeFile(artifactPath, `${JSON.stringify(artifact, null, 2)}\n`, 'utf8');
  }

  return artifact;
}

export async function loadSubscriptionManagerArtifact() {
  try {
    const raw = await fs.readFile(artifactPath, 'utf8');
    return JSON.parse(raw);
  } catch {
    return compileSubscriptionManager({ writeArtifact: true });
  }
}

export async function writeDeploymentRecord(networkName, record) {
  await fs.mkdir(deploymentsDir, { recursive: true });
  const filePath = path.join(deploymentsDir, `${networkName}.json`);
  await fs.writeFile(filePath, `${JSON.stringify(record, null, 2)}\n`, 'utf8');
  return filePath;
}
