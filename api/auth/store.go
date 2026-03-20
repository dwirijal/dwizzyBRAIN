package authapi

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func scanUserRow(row interface{ Scan(dest ...any) error }) (userRecord, error) {
	var rec userRecord
	if err := row.Scan(
		&rec.ID,
		&rec.Username,
		&rec.DisplayName,
		&rec.AvatarURL,
		&rec.Timezone,
		&rec.Locale,
		&rec.Plan,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	); err != nil {
		return userRecord{}, err
	}
	return rec, nil
}

func scanSessionRow(row interface{ Scan(dest ...any) error }) (sessionRecord, error) {
	var rec sessionRecord
	if err := row.Scan(
		&rec.ID,
		&rec.UserID,
		&rec.Status,
		&rec.FamilyID,
		&rec.LastSeenAt,
		&rec.ExpiresAt,
		&rec.CreatedAt,
		&rec.RevokedAt,
	); err != nil {
		return sessionRecord{}, err
	}
	return rec, nil
}

func scanRefreshRow(row interface{ Scan(dest ...any) error }) (refreshRecord, error) {
	var rec refreshRecord
	if err := row.Scan(
		&rec.ID,
		&rec.SessionID,
		&rec.TokenHash,
		&rec.ConsumedAt,
		&rec.ExpiresAt,
		&rec.CreatedAt,
	); err != nil {
		return refreshRecord{}, err
	}
	return rec, nil
}

func (u userRecord) toProfile() UserProfile {
	return UserProfile{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		Timezone:    u.Timezone,
		Locale:      u.Locale,
		Plan:        u.Plan,
	}
}

func (s sessionRecord) toInfo() SessionInfo {
	return SessionInfo{
		ID:         s.ID,
		Status:     s.Status,
		FamilyID:   s.FamilyID,
		LastSeenAt: s.LastSeenAt,
		ExpiresAt:  s.ExpiresAt,
		CreatedAt:  s.CreatedAt,
		RevokedAt:  s.RevokedAt,
	}
}

func (s sessionRecord) withLastSeen(lastSeen time.Time) sessionRecord {
	s.LastSeenAt = lastSeen
	return s
}

func (s *Service) linkExistingDiscordIdentity(ctx context.Context, profile discordProfile, meta RequestMeta, now time.Time) (userRecord, error) {
	existing, err := s.userByDiscordID(ctx, profile.ID)
	if err != nil {
		return userRecord{}, err
	}
	return s.updateUserProfile(ctx, existing.ID, profile, meta, now)
}

func (s *Service) userByDiscordID(ctx context.Context, providerUserID string) (userRecord, error) {
	if s.db == nil {
		return userRecord{}, fmt.Errorf("postgres pool is required")
	}
	row := s.db.QueryRow(ctx, `
SELECT u.id, u.username, u.display_name, COALESCE(u.avatar_url, ''), COALESCE(u.timezone, 'UTC'), COALESCE(u.locale, 'id-ID'), COALESCE(u.plan_override, 'free'), u.created_at, u.updated_at
FROM auth_identities i
JOIN users u ON u.id = i.user_id
WHERE i.provider = $1 AND i.provider_user_id = $2`, discordProviderName, providerUserID)
	return scanUserRow(row)
}

func (s *Service) updateUserProfile(ctx context.Context, userID string, profile discordProfile, meta RequestMeta, now time.Time) (userRecord, error) {
	if s.db == nil {
		return userRecord{}, fmt.Errorf("postgres pool is required")
	}
	displayName := strings.TrimSpace(firstNonEmpty(profile.GlobalName, profile.Username))
	if displayName == "" {
		displayName = "Discord User"
	}
	avatarURL := discordAvatarURL(profile)
	locale := strings.TrimSpace(profile.Locale)
	if locale == "" {
		locale = "id-ID"
	}

	if _, err := s.db.Exec(ctx, `
UPDATE users
SET display_name = $2,
    avatar_url = $3,
    locale = $4,
    updated_at = $5
WHERE id = $1`,
		userID, displayName, avatarURL, locale, now,
	); err != nil {
		return userRecord{}, fmt.Errorf("update user profile: %w", err)
	}
	row := s.db.QueryRow(ctx, `
SELECT id, username, display_name, COALESCE(avatar_url, ''), COALESCE(timezone, 'UTC'), COALESCE(locale, 'id-ID'), COALESCE(plan_override, 'free'), created_at, updated_at
FROM users WHERE id = $1`, userID)
	return scanUserRow(row)
}
