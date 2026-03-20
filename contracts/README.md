# Subscription Manager Contracts

This package contains the EVM `SubscriptionManager.sol` contract and a minimal Node-based deployment toolchain for:

- Base Mainnet
- BNB Smart Chain Mainnet
- Kaia Mainnet
- Sonic Mainnet
- Arbitrum One
- Abstract Mainnet
- HyperEVM

## Contract model

The contract is intentionally small:

- `isSubscribed(address)` returns a boolean view
- owner-controlled subscription expiry per wallet
- no payment logic inside the contract

That keeps the contract deployable across multiple EVM networks while letting `dwizzyBRAIN` resolve plan entitlement from any of them.

## Environment

Copy `contracts/.env.example` to `contracts/.env` or export the same values in your shell.
Secrets can also be mounted as files via `*_FILE`, for example:

- `SUBSCRIPTION_DEPLOYER_PRIVATE_KEY_FILE=/run/secrets/subscription_deployer_key`

Required for deployment:

- `SUBSCRIPTION_DEPLOYER_PRIVATE_KEY`
- or `SUBSCRIPTION_DEPLOYER_PRIVATE_KEY_FILE`
- `SUBSCRIPTION_BASE_RPC_URL`
- `SUBSCRIPTION_BSC_RPC_URL`
- `SUBSCRIPTION_KAIA_RPC_URL`
- `SUBSCRIPTION_SONIC_RPC_URL`
- `SUBSCRIPTION_ARBITRUM_RPC_URL`
- `SUBSCRIPTION_ABSTRACT_RPC_URL`
- `SUBSCRIPTION_HYPEREVM_RPC_URL`

Optional:

- `SUBSCRIPTION_OWNER_ADDRESS`

## Commands

```bash
npm install
npm run compile
npm run deploy:base
npm run deploy:bsc
npm run deploy:kaia
npm run deploy:sonic
npm run deploy:arbitrum
npm run deploy:abstract
npm run deploy:hyperevm
npm run deploy:all
```

Each deploy writes a record into `contracts/deployments/<network>.json`.

## Backend wiring

Once deployed, set `SUBSCRIPTION_NETWORKS` in `dwizzyBRAIN/.env` using:

```text
name|chain_id|rpc_url|contract_address[|method]
```

Example:

```text
base|8453|https://mainnet.base.org|0x...;bsc|56|https://bsc-dataseed.binance.org|0x...;hyperevm|999|https://rpc.hyperliquid.xyz/evm|0x...
```
