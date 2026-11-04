package repository

import (
	"context"
	"errors"

	"github.com/XiaomaiTX/go-react-starter/internal/model"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("记录不存在")

type ExampleRepository interface {
	List(context.Context, int, int) ([]model.Example, int64, error)
	Get(context.Context, int64) (model.Example, error)
	Create(context.Context, model.Example) (model.Example, error)
	Update(context.Context, model.Example) (model.Example, error)
	Delete(context.Context, int64) error
}

type exampleRepository struct{ db *gorm.DB }

func NewExampleRepository(db *gorm.DB) ExampleRepository { return &exampleRepository{db: db} }

func (r *exampleRepository) List(ctx context.Context, offset, limit int) ([]model.Example, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.Example{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var examples []model.Example
	err := r.db.WithContext(ctx).Order("created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&examples).Error
	return examples, total, err
}

func (r *exampleRepository) Get(ctx context.Context, id int64) (model.Example, error) {
	var example model.Example
	err := r.db.WithContext(ctx).First(&example, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Example{}, ErrNotFound
	}
	return example, err
}

func (r *exampleRepository) Create(ctx context.Context, example model.Example) (model.Example, error) {
	err := r.db.WithContext(ctx).Create(&example).Error
	return example, err
}

func (r *exampleRepository) Update(ctx context.Context, example model.Example) (model.Example, error) {
	result := r.db.WithContext(ctx).Model(&model.Example{}).Where("id = ?", example.ID).Updates(map[string]any{"name": example.Name, "description": example.Description})
	if result.Error != nil {
		return model.Example{}, result.Error
	}
	if result.RowsAffected == 0 {
		return model.Example{}, ErrNotFound
	}
	return r.Get(ctx, example.ID)
}

func (r *exampleRepository) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&model.Example{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
