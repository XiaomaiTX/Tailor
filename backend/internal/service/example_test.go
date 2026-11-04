package service

import (
	"context"
	"errors"
	"testing"

	"github.com/XiaomaiTX/go-react-starter/internal/model"
	"github.com/XiaomaiTX/go-react-starter/internal/repository"
)

type fakeRepository struct {
	item model.Example
	list []model.Example
	err  error
}

func (f *fakeRepository) List(context.Context, int, int) ([]model.Example, int64, error) {
	return f.list, int64(len(f.list)), f.err
}
func (f *fakeRepository) Get(context.Context, int64) (model.Example, error) { return f.item, f.err }
func (f *fakeRepository) Create(_ context.Context, item model.Example) (model.Example, error) {
	item.ID = 1
	return item, f.err
}
func (f *fakeRepository) Update(_ context.Context, item model.Example) (model.Example, error) {
	return item, f.err
}
func (f *fakeRepository) Delete(context.Context, int64) error { return f.err }

func TestCreateNormalizesInput(t *testing.T) {
	service := NewExampleService(&fakeRepository{})
	description := "  描述  "
	result, err := service.Create(context.Background(), ExampleInput{Name: "  名称  ", Description: &description})
	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "名称" || result.Description == nil || *result.Description != "描述" {
		t.Fatalf("归一化失败：%+v", result)
	}
}

func TestListRejectsInvalidPagination(t *testing.T) {
	service := NewExampleService(&fakeRepository{})
	if _, err := service.List(context.Background(), 0, 20); !errors.Is(err, ErrValidation) {
		t.Fatalf("错误为 %v", err)
	}
}

func TestGetMapsRepositoryNotFound(t *testing.T) {
	service := NewExampleService(&fakeRepository{err: repository.ErrNotFound})
	if _, err := service.Get(context.Background(), 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("错误为 %v", err)
	}
}
