package storageext

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"
)

const defaultStorageInterval = 24 * time.Hour

type Result struct {
	GDriveBackedUp bool
	R2Synced       bool
	BackupName     string
	BackupSource   string
	R2Source       string
}

type Job struct {
	gdrive           *GDriveBackup
	r2               *R2Sync
	backupSourcePath string
	backupName       string
	r2SourcePath     string
	interval         time.Duration
	logger           *log.Logger
}

func NewJob(gdrive *GDriveBackup, r2 *R2Sync, backupSourcePath, backupName, r2SourcePath string, interval time.Duration, logger *log.Logger) *Job {
	if interval <= 0 {
		interval = defaultStorageInterval
	}
	return &Job{
		gdrive:           gdrive,
		r2:               r2,
		backupSourcePath: strings.TrimSpace(backupSourcePath),
		backupName:       strings.TrimSpace(backupName),
		r2SourcePath:     strings.TrimSpace(r2SourcePath),
		interval:         interval,
		logger:           logger,
	}
}

func (j *Job) RunOnce(ctx context.Context) (Result, error) {
	if j == nil {
		return Result{}, fmt.Errorf("storage job is required")
	}

	backupSource := j.backupSourcePath
	r2Source := j.r2SourcePath

	if j.gdrive == nil && j.r2 == nil {
		return Result{}, fmt.Errorf("no storage bridges configured")
	}
	if j.gdrive != nil && backupSource == "" && j.r2 != nil && r2Source == "" {
		return Result{}, fmt.Errorf("storage bridge source path is required")
	}

	result := Result{}
	if j.gdrive != nil && backupSource != "" {
		backupName := j.backupName
		if backupName == "" {
			backupName = defaultBackupName(backupSource)
		}
		if err := j.gdrive.Backup(ctx, backupSource, backupName); err != nil {
			return Result{}, err
		}
		result.GDriveBackedUp = true
		result.BackupName = backupName
		result.BackupSource = backupSource
		if j.logger != nil {
			j.logger.Printf("storage gdrive backup source=%s name=%s", backupSource, backupName)
		}
	}
	if j.r2 != nil && r2Source != "" {
		if err := j.r2.SyncDirectory(ctx, r2Source); err != nil {
			return Result{}, err
		}
		result.R2Synced = true
		result.R2Source = r2Source
		if j.logger != nil {
			j.logger.Printf("storage r2 sync source=%s", r2Source)
		}
	}

	if !result.GDriveBackedUp && !result.R2Synced {
		return Result{}, fmt.Errorf("no storage bridge actions configured")
	}

	return result, nil
}

func (j *Job) Run(ctx context.Context) error {
	if _, err := j.RunOnce(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := j.RunOnce(ctx); err != nil {
				return err
			}
		}
	}
}

func defaultBackupName(sourcePath string) string {
	base := strings.TrimSpace(filepath.Base(strings.TrimRight(sourcePath, "/")))
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = "storage-backup"
	}
	return fmt.Sprintf("%s-%s", base, time.Now().UTC().Format("20060102-150405"))
}
