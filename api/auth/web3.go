package authapi

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type nonceRecord struct {
	ID            string
	WalletAddress string
	Nonce         string
	Purpose       string
	ExpiresAt     time.Time
	UsedAt        *time.Time
	CreatedAt     time.Time
}

func (s *Service) insertNonce(ctx context.Context, wallet, nonce, purpose string, expiresAt, now time.Time) error {
	if s.db == nil {
		return fmt.Errorf("postgres pool is required")
	}
	_, err := s.db.Exec(ctx, `
INSERT INTO auth_nonces (
    id, wallet_address, nonce, purpose, expires_at, created_at
) VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.NewString(), wallet, nonce, purpose, expiresAt, now,
	)
	if err != nil {
		return fmt.Errorf("insert nonce: %w", err)
	}
	return nil
}

func (s *Service) nonceRecord(ctx context.Context, wallet, nonce, purpose string) (nonceRecord, error) {
	if s.db == nil {
		return nonceRecord{}, fmt.Errorf("postgres pool is required")
	}
	row := s.db.QueryRow(ctx, `
SELECT id, wallet_address, nonce, purpose, expires_at, used_at, created_at
FROM auth_nonces
WHERE wallet_address = $1 AND nonce = $2 AND purpose = $3`,
		wallet, nonce, purpose,
	)
	var rec nonceRecord
	if err := row.Scan(&rec.ID, &rec.WalletAddress, &rec.Nonce, &rec.Purpose, &rec.ExpiresAt, &rec.UsedAt, &rec.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nonceRecord{}, ErrInvalidNonce
		}
		return nonceRecord{}, fmt.Errorf("query nonce: %w", err)
	}
	return rec, nil
}

func (s *Service) upsertWalletUserTx(ctx context.Context, tx pgx.Tx, walletAddress string, meta RequestMeta, now time.Time) (userRecord, error) {
	existing, err := s.userByWalletTx(ctx, tx, walletAddress)
	if err == nil {
		updated, err := s.updateWalletUserTx(ctx, tx, existing.ID, walletAddress, meta, now)
		if err != nil {
			return userRecord{}, err
		}
		return updated, nil
	}
	if err != pgx.ErrNoRows {
		return userRecord{}, err
	}
	return s.insertWalletUserTx(ctx, tx, walletAddress, meta, now)
}

func (s *Service) userByWalletTx(ctx context.Context, tx pgx.Tx, walletAddress string) (userRecord, error) {
	row := tx.QueryRow(ctx, `
SELECT u.id, u.username, u.display_name, COALESCE(u.avatar_url, ''), COALESCE(u.timezone, 'UTC'), COALESCE(u.locale, 'id-ID'), COALESCE(u.plan_override, 'free'), u.created_at, u.updated_at
FROM auth_identities i
JOIN users u ON u.id = i.user_id
WHERE i.provider = $1 AND i.provider_user_id = $2`,
		"evm", walletAddress,
	)
	rec, err := scanUserRow(row)
	if err == pgx.ErrNoRows {
		return userRecord{}, pgx.ErrNoRows
	}
	return rec, err
}

func (s *Service) updateWalletUserTx(ctx context.Context, tx pgx.Tx, userID, walletAddress string, meta RequestMeta, now time.Time) (userRecord, error) {
	displayName := walletDisplayName(walletAddress)
	if _, err := tx.Exec(ctx, `
UPDATE users
SET display_name = $2,
    updated_at = $3
WHERE id = $1`,
		userID, displayName, now,
	); err != nil {
		return userRecord{}, fmt.Errorf("update wallet user: %w", err)
	}
	row := tx.QueryRow(ctx, `
SELECT id, username, display_name, COALESCE(avatar_url, ''), COALESCE(timezone, 'UTC'), COALESCE(locale, 'id-ID'), COALESCE(plan_override, 'free'), created_at, updated_at
FROM users WHERE id = $1`, userID)
	return scanUserRow(row)
}

func (s *Service) insertWalletUserTx(ctx context.Context, tx pgx.Tx, walletAddress string, meta RequestMeta, now time.Time) (userRecord, error) {
	displayName := walletDisplayName(walletAddress)
	base := sanitizeUsername("wallet_" + strings.TrimPrefix(strings.ToLower(walletAddress), "0x"))
	if base == "" {
		base = "wallet_user"
	}
	email := strings.TrimPrefix(strings.ToLower(walletAddress), "0x") + "@wallet.local"
	for attempt := 0; attempt < 8; attempt++ {
		username := usernameCandidate(base, attempt)
		id := uuid.NewString()
		row := tx.QueryRow(ctx, `
INSERT INTO users (
    id, email, name, picture, username, display_name, avatar_url, timezone, locale, plan_override, created_at, updated_at
) VALUES ($1, $2, $3, NULL, $4, $5, NULL, 'UTC', 'id-ID', NULL, $6, $6)
ON CONFLICT (username) DO NOTHING
RETURNING id, username, display_name, COALESCE(avatar_url, ''), COALESCE(timezone, 'UTC'), COALESCE(locale, 'id-ID'), COALESCE(plan_override, 'free'), created_at, updated_at`,
			id, email, displayName, username, displayName, now,
		)
		user, err := scanUserRow(row)
		if err != nil {
			if err == pgx.ErrNoRows {
				continue
			}
			return userRecord{}, err
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO auth_identities (
    id, user_id, provider, provider_user_id, metadata_json, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $6)`,
			uuid.NewString(), user.ID, "evm", walletAddress, walletMetadataJSON(walletAddress, meta), now,
		); err != nil {
			return userRecord{}, fmt.Errorf("insert wallet identity: %w", err)
		}
		return user, nil
	}
	return userRecord{}, fmt.Errorf("unable to allocate username for wallet %s", walletAddress)
}

func buildNonceChallenge(wallet, purpose, nonce string, createdAt time.Time) string {
	return strings.Join([]string{
		"dwizzyBRAIN wants you to sign in with Ethereum",
		"Wallet: " + wallet,
		"Purpose: " + purpose,
		"Nonce: " + nonce,
		"Issued At: " + createdAt.UTC().Format(time.RFC3339),
	}, "\n")
}

func normalizeWalletAddress(raw string) (string, error) {
	wallet := strings.ToLower(strings.TrimSpace(raw))
	if wallet == "" {
		return "", ErrInvalidWalletAddress
	}
	if !common.IsHexAddress(wallet) {
		return "", ErrInvalidWalletAddress
	}
	return strings.ToLower(common.HexToAddress(wallet).Hex()), nil
}

func normalizeNoncePurpose(purpose string) string {
	if purpose == "" {
		return "login"
	}
	purpose = strings.ToLower(strings.TrimSpace(purpose))
	if purpose != "login" {
		return "login"
	}
	return purpose
}

func walletDisplayName(wallet string) string {
	wallet = strings.ToLower(strings.TrimSpace(wallet))
	if len(wallet) < 10 {
		return wallet
	}
	return wallet[:8] + "…" + wallet[len(wallet)-4:]
}

func walletMetadataJSON(wallet string, meta RequestMeta) map[string]any {
	return map[string]any{
		"wallet": map[string]any{
			"address": wallet,
			"purpose": "login",
		},
		"request": map[string]any{
			"remote_addr": meta.RemoteAddr,
			"user_agent":  meta.UserAgent,
		},
	}
}

func recoverWalletFromSignature(message, signature string) (string, error) {
	sig, err := decodeSignature(signature)
	if err != nil {
		return "", err
	}
	if sig[64] >= 27 {
		sig[64] -= 27
	}
	if sig[64] > 1 {
		return "", ErrInvalidSignature
	}

	hash := accounts.TextHash([]byte(message))
	pub, err := ethcrypto.SigToPub(hash, sig)
	if err != nil {
		return "", ErrInvalidSignature
	}
	addr := ethcrypto.PubkeyToAddress(*pub).Hex()
	return strings.ToLower(addr), nil
}

func decodeSignature(signature string) ([]byte, error) {
	cleaned := strings.TrimSpace(signature)
	cleaned = strings.TrimPrefix(cleaned, "0x")
	raw, err := hex.DecodeString(cleaned)
	if err != nil {
		return nil, ErrInvalidSignature
	}
	if len(raw) != 65 {
		return nil, ErrInvalidSignature
	}
	return raw, nil
}
