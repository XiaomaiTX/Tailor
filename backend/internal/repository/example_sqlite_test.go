package repository

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/XiaomaiTX/go-react-starter/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExampleRepositoryWithTemporarySQLite(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "examples.db")
	db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDatabase, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDatabase.Close() //nolint:errcheck

	if err := db.AutoMigrate(&model.Example{}); err != nil {
		t.Fatal(err)
	}
	repository := NewExampleRepository(db)
	description := "临时数据库"
	created, err := repository.Create(ctx, model.Example{Name: "Example", Description: &description})
	if err != nil || created.ID < 1 {
		t.Fatalf("创建失败：%v，%+v", err, created)
	}
	list, total, err := repository.List(ctx, 0, 20)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("查询失败：%v，total=%d，list=%d", err, total, len(list))
	}
}
