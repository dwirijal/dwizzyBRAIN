package storageext

import (
	"context"
	"fmt"
	"strings"
)

type GDriveBackup struct {
	rclone RcloneRunner
	remote string
	prefix string
}

func NewGDriveBackup(rclone RcloneRunner, remote, prefix string) *GDriveBackup {
	return &GDriveBackup{
		rclone: rclone,
		remote: strings.TrimSpace(remote),
		prefix: strings.Trim(strings.TrimSpace(prefix), "/"),
	}
}

func (b *GDriveBackup) Enabled() bool {
	return b != nil && b.remote != ""
}

func (b *GDriveBackup) Backup(ctx context.Context, sourcePath, backupName string) error {
	if !b.Enabled() {
		return fmt.Errorf("google drive backup is not configured")
	}
	sourcePath = strings.TrimSpace(sourcePath)
	backupName = strings.TrimSpace(backupName)
	if sourcePath == "" {
		return fmt.Errorf("source path is required")
	}
	if backupName == "" {
		return fmt.Errorf("backup name is required")
	}
	return b.rclone.Run(ctx, "copy", sourcePath, b.remotePath(backupName), "--metadata")
}

func (b *GDriveBackup) UploadFile(ctx context.Context, sourcePath, remoteFilePath string) error {
	if !b.Enabled() {
		return fmt.Errorf("google drive backup is not configured")
	}
	sourcePath = strings.TrimSpace(sourcePath)
	remoteFilePath = strings.Trim(strings.TrimSpace(remoteFilePath), "/")
	if sourcePath == "" {
		return fmt.Errorf("source path is required")
	}
	if remoteFilePath == "" {
		return fmt.Errorf("remote file path is required")
	}
	return b.rclone.Run(ctx, "copyto", sourcePath, b.remoteFilePath(remoteFilePath), "--metadata")
}

func (b *GDriveBackup) ShareLink(ctx context.Context, remoteFilePath string) (string, error) {
	if !b.Enabled() {
		return "", fmt.Errorf("google drive backup is not configured")
	}
	remoteFilePath = strings.Trim(strings.TrimSpace(remoteFilePath), "/")
	if remoteFilePath == "" {
		return "", fmt.Errorf("remote file path is required")
	}
	return b.rclone.Output(ctx, "link", b.remoteFilePath(remoteFilePath))
}

func (b *GDriveBackup) DeleteFile(ctx context.Context, remoteFilePath string) error {
	if !b.Enabled() {
		return fmt.Errorf("google drive backup is not configured")
	}
	remoteFilePath = strings.Trim(strings.TrimSpace(remoteFilePath), "/")
	if remoteFilePath == "" {
		return fmt.Errorf("remote file path is required")
	}
	return b.rclone.Delete(ctx, "deletefile", b.remoteFilePath(remoteFilePath))
}

func (b *GDriveBackup) remotePath(segment string) string {
	segment = strings.Trim(strings.TrimSpace(segment), "/")
	if b.prefix == "" {
		if segment == "" {
			return b.remote + ":"
		}
		return b.remote + ":" + segment
	}
	if segment == "" {
		return b.remote + ":" + b.prefix
	}
	return b.remote + ":" + b.prefix + "/" + segment
}

func (b *GDriveBackup) remoteFilePath(filePath string) string {
	filePath = strings.Trim(strings.TrimSpace(filePath), "/")
	if filePath == "" {
		return b.remotePath("")
	}
	return b.remotePath(filePath)
}
