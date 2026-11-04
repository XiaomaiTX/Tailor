//go:build integration

package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/XiaomaiTX/go-react-starter/internal/model"
	migrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestExampleRepositoryAndMigrationWithTemporaryPostgres(t *testing.T) {
	ctx := context.Background()
	container, err := postgrescontainer.Run(ctx, "postgres:17.6-alpine3.22",
		postgrescontainer.WithDatabase("starter"),
		postgrescontainer.WithUsername("starter"),
		postgrescontainer.WithPassword("starter"),
	)
	if err != nil {
		t.Skipf("无法启动临时 PostgreSQL：%v", err)
	}
	defer container.Terminate(ctx) //nolint:errcheck

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	migration, err := migrate.New("file://../../migrations", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := migration.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatal(err)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
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
	if err := migration.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatal(err)
	}
}
