// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract SubscriptionManager {
    struct Subscription {
        uint64 expiresAt;
        uint64 updatedAt;
        address updatedBy;
    }

    address public owner;

    mapping(address => Subscription) private subscriptions;

    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);
    event SubscriptionUpdated(address indexed wallet, uint64 expiresAt, address indexed updatedBy);
    event SubscriptionRevoked(address indexed wallet, address indexed updatedBy);

    modifier onlyOwner() {
        require(msg.sender == owner, "SubscriptionManager: not owner");
        _;
    }

    constructor(address initialOwner) {
        require(initialOwner != address(0), "SubscriptionManager: owner required");
        owner = initialOwner;
        emit OwnershipTransferred(address(0), initialOwner);
    }

    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "SubscriptionManager: owner required");
        address previousOwner = owner;
        owner = newOwner;
        emit OwnershipTransferred(previousOwner, newOwner);
    }

    function setSubscription(address wallet, uint64 expiresAt) external onlyOwner {
        require(wallet != address(0), "SubscriptionManager: wallet required");
        subscriptions[wallet] = Subscription({
            expiresAt: expiresAt,
            updatedAt: uint64(block.timestamp),
            updatedBy: msg.sender
        });
        emit SubscriptionUpdated(wallet, expiresAt, msg.sender);
    }

    function setSubscriptions(address[] calldata wallets, uint64[] calldata expiresAts) external onlyOwner {
        require(wallets.length == expiresAts.length, "SubscriptionManager: length mismatch");
        for (uint256 i = 0; i < wallets.length; i++) {
            require(wallets[i] != address(0), "SubscriptionManager: wallet required");
            subscriptions[wallets[i]] = Subscription({
                expiresAt: expiresAts[i],
                updatedAt: uint64(block.timestamp),
                updatedBy: msg.sender
            });
            emit SubscriptionUpdated(wallets[i], expiresAts[i], msg.sender);
        }
    }

    function revokeSubscription(address wallet) external onlyOwner {
        require(wallet != address(0), "SubscriptionManager: wallet required");
        delete subscriptions[wallet];
        emit SubscriptionRevoked(wallet, msg.sender);
    }

    function isSubscribed(address wallet) external view returns (bool) {
        return subscriptions[wallet].expiresAt > block.timestamp;
    }

    function subscriptionOf(address wallet)
        external
        view
        returns (uint64 expiresAt, bool active, uint64 updatedAt, address updatedBy)
    {
        Subscription memory sub = subscriptions[wallet];
        return (sub.expiresAt, sub.expiresAt > block.timestamp, sub.updatedAt, sub.updatedBy);
    }
}
