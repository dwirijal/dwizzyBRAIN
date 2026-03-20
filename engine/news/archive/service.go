package archive

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	storageext "dwizzyBRAIN/engine/storage_ext"
)

type uploader interface {
	UploadFile(ctx context.Context, sourcePath, remoteFilePath string) error
	ShareLink(ctx context.Context, remoteFilePath string) (string, error)
}

type store interface {
	ListPendingArticles(ctx context.Context, limit int) ([]Article, error)
	LoadMetadata(ctx context.Context, articleID int64) (*Metadata, error)
	LoadEntities(ctx context.Context, articleID int64) ([]Entity, error)
	UpsertExport(ctx context.Context, rec ExportRecord) error
}

type Service struct {
	store       store
	uploader    uploader
	remoteBase  string
	limit       int
	now         func() time.Time
	tempDirFunc func() string
}

func NewService(store store, uploader uploader, remoteBase string, limit int) *Service {
	if limit <= 0 {
		limit = 20
	}
	return &Service{
		store:       store,
		uploader:    uploader,
		remoteBase:  strings.Trim(strings.TrimSpace(remoteBase), "/"),
		limit:       limit,
		now:         time.Now,
		tempDirFunc: os.TempDir,
	}
}

func (s *Service) RunOnce(ctx context.Context) (Result, error) {
	if s.store == nil {
		return Result{}, fmt.Errorf("news archive store is required")
	}
	if s.uploader == nil {
		return Result{}, fmt.Errorf("news archive uploader is required")
	}
	if s.remoteBase == "" {
		return Result{}, fmt.Errorf("news archive remote base is required")
	}

	articles, err := s.store.ListPendingArticles(ctx, s.limit)
	if err != nil {
		return Result{}, err
	}

	result := Result{ArticlesScanned: len(articles)}
	for _, article := range articles {
		if err := s.exportArticle(ctx, article); err != nil {
			result.Failures++
			result.FailedArticles = append(result.FailedArticles, article.ID)
			continue
		}
		result.ArticlesExported++
	}

	return result, nil
}

func (s *Service) exportArticle(ctx context.Context, article Article) error {
	if article.Metadata == nil {
		meta, err := s.store.LoadMetadata(ctx, article.ID)
		if err != nil {
			return err
		}
		article.Metadata = meta
	}
	if len(article.Entities) == 0 {
		entities, err := s.store.LoadEntities(ctx, article.ID)
		if err != nil {
			return err
		}
		article.Entities = entities
	}

	exportedAt := s.now().UTC()
	md := RenderMarkdown(article)
	year, month := article.PublishedYearMonth()
	folderName := fmt.Sprintf("%d-%s", article.ID, Slugify(article.Title))
	fileName := "content.md"
	remoteFolderPath := filepath.ToSlash(filepath.Join(
		s.remoteBase,
		"articles",
		strings.ToLower(strings.TrimSpace(article.Source)),
		year,
		month,
		folderName,
	))
	markdownRemotePath := filepath.ToSlash(filepath.Join(remoteFolderPath, fileName))

	tmpDir, err := os.MkdirTemp(s.tempDirFunc(), "dwizzy-news-md-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	markdownLocalPath := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(markdownLocalPath, []byte(md), 0o600); err != nil {
		return fmt.Errorf("write markdown file: %w", err)
	}

	if err := s.uploader.UploadFile(ctx, markdownLocalPath, markdownRemotePath); err != nil {
		return err
	}
	link, err := s.uploader.ShareLink(ctx, markdownRemotePath)
	if err != nil {
		return err
	}

	rec := ExportRecord{
		ArticleID:         article.ID,
		Title:             article.Title,
		DriveURL:          strings.TrimSpace(link),
		DrivePath:         markdownRemotePath,
		FileName:          fileName,
		ContentFolderPath: remoteFolderPath,
		ContentJSONPath:   "",
		ContentJSONURL:    "",
		ExportedAt:        exportedAt,
	}
	if err := s.store.UpsertExport(ctx, rec); err != nil {
		return err
	}
	return nil
}

func NewDefaultUploader(remote *storageext.GDriveBackup) uploader {
	return remote
}
