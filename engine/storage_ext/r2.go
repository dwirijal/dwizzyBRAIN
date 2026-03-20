package storageext

import (
	"context"
	"fmt"
	"strings"
)

type R2Sync struct {
	rclone RcloneRunner
	remote string
	prefix string
}

func NewR2Sync(rclone RcloneRunner, remote, prefix string) *R2Sync {
	return &R2Sync{
		rclone: rclone,
		remote: strings.TrimSpace(remote),
		prefix: strings.Trim(strings.TrimSpace(prefix), "/"),
	}
}

func (s *R2Sync) Enabled() bool {
	return s != nil && s.remote != ""
}

func (s *R2Sync) SyncDirectory(ctx context.Context, sourcePath string) error {
	if !s.Enabled() {
		return fmt.Errorf("cloudflare r2 sync is not configured")
	}
	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		return fmt.Errorf("source path is required")
	}
	dest := s.remote
	if s.prefix != "" {
		dest += ":" + s.prefix
	}
	return s.rclone.Run(ctx,
		"sync", sourcePath, dest,
		"--metadata",
	)
}
