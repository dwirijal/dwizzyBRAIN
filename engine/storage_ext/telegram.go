package storageext

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	redis "github.com/redis/go-redis/v9"

	sharedconfig "dwizzyBRAIN/shared/config"
)

var ErrTelegramNotConfigured = errors.New("telegram bridge is not configured")

type TelegramFileRecord struct {
	ID             int64
	FileKey        string
	FileID         string
	FileUniqueID   string
	MessageID      int64
	ChannelID      string
	FileType       string
	FileName       string
	FileSizeBytes  int64
	MimeType       string
	CoinID         string
	Timeframe      string
	DateContext    string
	UploadedAt     time.Time
	LastAccessedAt *time.Time
	AccessCount    int
}

type TelegramUploadRequest struct {
	FileKey       string
	FileName      string
	ContentType   string
	Caption       string
	FileType      string
	CoinID        string
	Timeframe     string
	DateContext   string
	ChannelID     string
	SendAsPhoto   bool
	Content       io.Reader
	FileSizeBytes int64
}

type telegramMediaRecord struct {
	FileID       string
	FileUniqueID string
	FileName     string
	FileSize     int64
	MimeType     string
}

type TelegramBridge struct {
	db        *pgxpool.Pool
	cache     redis.Cmdable
	http      *http.Client
	botToken  string
	channelID string
	baseURL   string
}

func NewTelegramBridge(db *pgxpool.Pool, cache redis.Cmdable) (*TelegramBridge, error) {
	botToken, err := sharedconfig.ReadOptional("TELEGRAM_BOT_TOKEN")
	if err != nil {
		return nil, err
	}
	return &TelegramBridge{
		db:        db,
		cache:     cache,
		http:      &http.Client{Timeout: 30 * time.Second},
		baseURL:   "https://api.telegram.org",
		botToken:  botToken,
		channelID: strings.TrimSpace(os.Getenv("TELEGRAM_FILES_CHANNEL")),
	}, nil
}

func (b *TelegramBridge) Enabled() bool {
	return b != nil && b.db != nil
}

func (b *TelegramBridge) CacheKey(fileKey string) string {
	return "telegram:file:" + strings.TrimSpace(fileKey)
}

func (b *TelegramBridge) Get(ctx context.Context, fileKey string) (TelegramFileRecord, error) {
	fileKey = strings.TrimSpace(fileKey)
	if fileKey == "" {
		return TelegramFileRecord{}, fmt.Errorf("file key is required")
	}

	if b.cache != nil {
		if raw, err := b.cache.Get(ctx, b.CacheKey(fileKey)).Result(); err == nil && raw != "" {
			var rec TelegramFileRecord
			if err := json.Unmarshal([]byte(raw), &rec); err == nil && rec.FileKey != "" {
				return rec, nil
			}
		}
	}

	if !b.Enabled() {
		return TelegramFileRecord{}, ErrTelegramNotConfigured
	}

	row := b.db.QueryRow(ctx, `
SELECT id, file_key, file_id, COALESCE(file_unique_id, ''), COALESCE(message_id, 0), COALESCE(channel_id, ''), file_type::text,
       COALESCE(file_name, ''), COALESCE(file_size_bytes, 0), COALESCE(mime_type, ''), COALESCE(coin_id, ''), COALESCE(timeframe, ''),
       COALESCE(date_context::text, ''), uploaded_at, last_accessed_at, access_count
FROM telegram_file_cache
WHERE file_key = $1`, fileKey)

	rec, err := scanTelegramFileRecord(row)
	if err != nil {
		return TelegramFileRecord{}, err
	}
	_ = b.mirror(ctx, rec)
	return rec, nil
}

func (b *TelegramBridge) Put(ctx context.Context, rec TelegramFileRecord) error {
	if !b.Enabled() {
		return ErrTelegramNotConfigured
	}
	rec.FileKey = strings.TrimSpace(rec.FileKey)
	rec.FileID = strings.TrimSpace(rec.FileID)
	rec.ChannelID = strings.TrimSpace(rec.ChannelID)
	rec.FileType = normalizeTelegramFileType(rec.FileType)
	if rec.FileKey == "" || rec.FileID == "" {
		return fmt.Errorf("file_key and file_id are required")
	}
	if rec.ChannelID == "" {
		rec.ChannelID = b.channelID
	}
	if rec.ChannelID == "" {
		return fmt.Errorf("channel_id is required")
	}
	if rec.UploadedAt.IsZero() {
		rec.UploadedAt = time.Now().UTC()
	}

	_, err := b.db.Exec(ctx, `
INSERT INTO telegram_file_cache (
    id, file_key, file_id, file_unique_id, message_id, channel_id, file_type, file_name, file_size_bytes, mime_type, coin_id, timeframe, date_context, uploaded_at, last_accessed_at, access_count
) VALUES (
    COALESCE(NULLIF($1, 0), nextval(pg_get_serial_sequence('telegram_file_cache', 'id'))),
    $2, $3, NULLIF($4, ''), NULLIF($5, 0), NULLIF($6, ''), $7::telegram_file_type, NULLIF($8, ''), NULLIF($9, 0), NULLIF($10, ''), NULLIF($11, ''), NULLIF($12, ''),
    NULLIF($13, '')::date, COALESCE($14, NOW()), $15, COALESCE($16, 0)
)
ON CONFLICT (file_key) DO UPDATE SET
    file_id = EXCLUDED.file_id,
    file_unique_id = COALESCE(EXCLUDED.file_unique_id, telegram_file_cache.file_unique_id),
    message_id = COALESCE(EXCLUDED.message_id, telegram_file_cache.message_id),
    channel_id = COALESCE(EXCLUDED.channel_id, telegram_file_cache.channel_id),
    file_type = EXCLUDED.file_type,
    file_name = COALESCE(EXCLUDED.file_name, telegram_file_cache.file_name),
    file_size_bytes = COALESCE(EXCLUDED.file_size_bytes, telegram_file_cache.file_size_bytes),
    mime_type = COALESCE(EXCLUDED.mime_type, telegram_file_cache.mime_type),
    coin_id = COALESCE(EXCLUDED.coin_id, telegram_file_cache.coin_id),
    timeframe = COALESCE(EXCLUDED.timeframe, telegram_file_cache.timeframe),
    date_context = COALESCE(EXCLUDED.date_context, telegram_file_cache.date_context),
    uploaded_at = COALESCE(EXCLUDED.uploaded_at, telegram_file_cache.uploaded_at),
    last_accessed_at = COALESCE(EXCLUDED.last_accessed_at, telegram_file_cache.last_accessed_at),
    access_count = COALESCE(EXCLUDED.access_count, telegram_file_cache.access_count)`,
		rec.ID, rec.FileKey, rec.FileID, rec.FileUniqueID, rec.MessageID, rec.ChannelID, rec.FileType, rec.FileName, rec.FileSizeBytes, rec.MimeType,
		rec.CoinID, rec.Timeframe, rec.DateContext, rec.UploadedAt, rec.LastAccessedAt, rec.AccessCount,
	)
	if err != nil {
		return fmt.Errorf("upsert telegram file cache: %w", err)
	}
	return b.mirror(ctx, rec)
}

func (b *TelegramBridge) Touch(ctx context.Context, fileKey string) error {
	if !b.Enabled() {
		return ErrTelegramNotConfigured
	}
	_, err := b.db.Exec(ctx, `
UPDATE telegram_file_cache
SET last_accessed_at = NOW(),
    access_count = access_count + 1
WHERE file_key = $1`, strings.TrimSpace(fileKey))
	if err != nil {
		return fmt.Errorf("touch telegram file cache: %w", err)
	}
	return nil
}

func (b *TelegramBridge) UploadDocument(ctx context.Context, req TelegramUploadRequest) (TelegramFileRecord, error) {
	if !b.Enabled() || b.botToken == "" {
		return TelegramFileRecord{}, ErrTelegramNotConfigured
	}
	if req.Content == nil {
		return TelegramFileRecord{}, fmt.Errorf("content is required")
	}
	channelID := strings.TrimSpace(req.ChannelID)
	if channelID == "" {
		channelID = b.channelID
	}
	if channelID == "" {
		return TelegramFileRecord{}, fmt.Errorf("channel id is required")
	}
	telegramResp, err := b.sendMedia(ctx, channelID, req, "document")
	if err != nil {
		return TelegramFileRecord{}, err
	}
	media := telegramMediaFromResponse(telegramResp, false)
	rec := TelegramFileRecord{
		FileKey:       req.FileKey,
		FileID:        media.FileID,
		FileUniqueID:  media.FileUniqueID,
		MessageID:     telegramResp.Result.MessageID,
		ChannelID:     channelID,
		FileType:      normalizeTelegramFileType(req.FileType),
		FileName:      firstNonEmpty(req.FileName, media.FileName),
		FileSizeBytes: media.FileSize,
		MimeType:      firstNonEmpty(req.ContentType, media.MimeType),
		CoinID:        req.CoinID,
		Timeframe:     req.Timeframe,
		DateContext:   req.DateContext,
		UploadedAt:    time.Now().UTC(),
		AccessCount:   0,
	}
	if err := b.Put(ctx, rec); err != nil {
		return TelegramFileRecord{}, err
	}
	return rec, nil
}

func (b *TelegramBridge) UploadPhoto(ctx context.Context, req TelegramUploadRequest) (TelegramFileRecord, error) {
	req.SendAsPhoto = true
	if !b.Enabled() || b.botToken == "" {
		return TelegramFileRecord{}, ErrTelegramNotConfigured
	}
	if req.Content == nil {
		return TelegramFileRecord{}, fmt.Errorf("content is required")
	}
	channelID := strings.TrimSpace(req.ChannelID)
	if channelID == "" {
		channelID = b.channelID
	}
	if channelID == "" {
		return TelegramFileRecord{}, fmt.Errorf("channel id is required")
	}
	telegramResp, err := b.sendMedia(ctx, channelID, req, "photo")
	if err != nil {
		return TelegramFileRecord{}, err
	}
	media := telegramMediaFromResponse(telegramResp, true)
	rec := TelegramFileRecord{
		FileKey:       req.FileKey,
		FileID:        media.FileID,
		FileUniqueID:  media.FileUniqueID,
		MessageID:     telegramResp.Result.MessageID,
		ChannelID:     channelID,
		FileType:      normalizeTelegramFileType(req.FileType),
		FileName:      firstNonEmpty(req.FileName, media.FileName),
		FileSizeBytes: media.FileSize,
		MimeType:      firstNonEmpty(req.ContentType, media.MimeType),
		CoinID:        req.CoinID,
		Timeframe:     req.Timeframe,
		DateContext:   req.DateContext,
		UploadedAt:    time.Now().UTC(),
		AccessCount:   0,
	}
	if err := b.Put(ctx, rec); err != nil {
		return TelegramFileRecord{}, err
	}
	return rec, nil
}

func (b *TelegramBridge) sendMedia(ctx context.Context, channelID string, req TelegramUploadRequest, fieldName string) (telegramSendResponse, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("chat_id", channelID); err != nil {
		return telegramSendResponse{}, fmt.Errorf("write chat id: %w", err)
	}
	if req.Caption != "" {
		if err := writer.WriteField("caption", req.Caption); err != nil {
			return telegramSendResponse{}, fmt.Errorf("write caption: %w", err)
		}
	}
	part, err := writer.CreateFormFile(fieldName, nonEmptyFilename(req.FileName, fieldName))
	if err != nil {
		return telegramSendResponse{}, fmt.Errorf("create upload part: %w", err)
	}
	if _, err := io.Copy(part, req.Content); err != nil {
		return telegramSendResponse{}, fmt.Errorf("stream upload content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return telegramSendResponse{}, fmt.Errorf("close multipart body: %w", err)
	}

	endpoint := strings.TrimRight(b.baseURL, "/") + "/bot" + b.botToken + "/sendDocument"
	if req.SendAsPhoto {
		endpoint = strings.TrimRight(b.baseURL, "/") + "/bot" + b.botToken + "/sendPhoto"
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return telegramSendResponse{}, fmt.Errorf("create telegram request: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := b.http.Do(httpReq)
	if err != nil {
		return telegramSendResponse{}, fmt.Errorf("send telegram media: %w", err)
	}
	defer resp.Body.Close()

	var payload telegramSendResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return telegramSendResponse{}, fmt.Errorf("decode telegram response: %w", err)
	}
	if !payload.OK {
		return telegramSendResponse{}, fmt.Errorf("telegram upload failed")
	}
	return payload, nil
}

func telegramMediaFromResponse(resp telegramSendResponse, photo bool) telegramMediaRecord {
	if photo {
		if n := len(resp.Result.Photo); n > 0 {
			media := resp.Result.Photo[n-1]
			return telegramMediaRecord{
				FileID:       media.FileID,
				FileUniqueID: media.FileUniqueID,
				FileSize:     media.FileSize,
			}
		}
		return telegramMediaRecord{}
	}
	return telegramMediaRecord{
		FileID:       resp.Result.Document.FileID,
		FileUniqueID: resp.Result.Document.FileUniqueID,
		FileName:     resp.Result.Document.FileName,
		FileSize:     resp.Result.Document.FileSize,
		MimeType:     resp.Result.Document.MimeType,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (b *TelegramBridge) mirror(ctx context.Context, rec TelegramFileRecord) error {
	if b.cache == nil {
		return nil
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal telegram file cache: %w", err)
	}
	if err := b.cache.Set(ctx, b.CacheKey(rec.FileKey), data, 0).Err(); err != nil {
		return fmt.Errorf("mirror telegram file cache: %w", err)
	}
	return nil
}

func normalizeTelegramFileType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "chart", "csv_export", "backup_db", "backup_valkey", "report":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "other"
	}
}

func nonEmptyFilename(name, fallback string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	fallback = strings.TrimSpace(fallback)
	if fallback != "" {
		return fallback
	}
	return uuid.NewString()
}

type telegramSendResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		MessageID int64 `json:"message_id"`
		Document  struct {
			FileID       string `json:"file_id"`
			FileUniqueID string `json:"file_unique_id"`
			FileName     string `json:"file_name"`
			FileSize     int64  `json:"file_size"`
			MimeType     string `json:"mime_type"`
		} `json:"document"`
		Photo []struct {
			FileID       string `json:"file_id"`
			FileUniqueID string `json:"file_unique_id"`
			FileSize     int64  `json:"file_size"`
			Width        int64  `json:"width"`
			Height       int64  `json:"height"`
		} `json:"photo"`
	} `json:"result"`
}

func scanTelegramFileRecord(row pgx.Row) (TelegramFileRecord, error) {
	var rec TelegramFileRecord
	var lastAccessedAt *time.Time
	if err := row.Scan(
		&rec.ID,
		&rec.FileKey,
		&rec.FileID,
		&rec.FileUniqueID,
		&rec.MessageID,
		&rec.ChannelID,
		&rec.FileType,
		&rec.FileName,
		&rec.FileSizeBytes,
		&rec.MimeType,
		&rec.CoinID,
		&rec.Timeframe,
		&rec.DateContext,
		&rec.UploadedAt,
		&lastAccessedAt,
		&rec.AccessCount,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TelegramFileRecord{}, err
		}
		return TelegramFileRecord{}, fmt.Errorf("scan telegram file cache: %w", err)
	}
	rec.LastAccessedAt = lastAccessedAt
	return rec, nil
}
