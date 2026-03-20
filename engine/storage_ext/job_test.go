package storageext

import (
	"context"
	"testing"
)

func TestJobRunOnce(t *testing.T) {
	gdriveRunner := RcloneRunner{Bin: "rclone", Exec: &fakeCommandRunner{}}
	r2Runner := RcloneRunner{Bin: "rclone", Exec: &fakeCommandRunner{}}
	job := NewJob(
		NewGDriveBackup(gdriveRunner, "gdrive", "backups"),
		NewR2Sync(r2Runner, "r2", "assets"),
		"/tmp/db.dump",
		"pg_dump-20260319",
		"/tmp/assets",
		0,
		nil,
	)

	result, err := job.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() returned error: %v", err)
	}
	if !result.GDriveBackedUp || !result.R2Synced {
		t.Fatalf("expected both actions to run, got %+v", result)
	}

	gdriveFake := gdriveRunner.Exec.(*fakeCommandRunner)
	if len(gdriveFake.args) < 4 || gdriveFake.args[0] != "copy" || gdriveFake.args[1] != "/tmp/db.dump" || gdriveFake.args[2] != "gdrive:backups/pg_dump-20260319" {
		t.Fatalf("unexpected gdrive args: %#v", gdriveFake.args)
	}

	r2Fake := r2Runner.Exec.(*fakeCommandRunner)
	if len(r2Fake.args) < 4 || r2Fake.args[0] != "sync" || r2Fake.args[1] != "/tmp/assets" || r2Fake.args[2] != "r2:assets" {
		t.Fatalf("unexpected r2 args: %#v", r2Fake.args)
	}
}

func TestJobRunOnceNoActions(t *testing.T) {
	job := NewJob(nil, nil, "", "", "", 0, nil)
	if _, err := job.RunOnce(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
