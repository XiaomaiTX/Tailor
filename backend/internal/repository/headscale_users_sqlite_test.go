package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/XiaomaiTX/go-react-starter/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSyncSnapshotRemovesUsersMissingFromSuccessfulSnapshot(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "headscale-users.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.HeadscaleUser{}, &model.Device{}); err != nil {
		t.Fatal(err)
	}
	repo := NewPeerRelayRepository(database)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := database.Create(&[]model.HeadscaleUser{
		{ID: "current-user", Username: "alice", Email: "alice@example.test", SyncedAt: now},
		{ID: "removed-user", Username: "bob", Email: "bob@example.test", SyncedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.SyncSnapshot(ctx, []model.HeadscaleUser{
		{ID: "current-user", Username: "alice", Email: "alice@example.test", SyncedAt: now.Add(time.Minute)},
	}, nil); err != nil {
		t.Fatal(err)
	}
	users, err := repo.Users(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].ID != "current-user" {
		t.Fatalf("current Headscale snapshot users=%+v", users)
	}
}
