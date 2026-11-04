package service

import (
	"context"
	"errors"
	"strings"

	"github.com/XiaomaiTX/go-react-starter/internal/model"
	"github.com/XiaomaiTX/go-react-starter/internal/repository"
)

var (
	ErrValidation = errors.New("请求不合法")
	ErrNotFound   = errors.New("资源不存在")
)

type ExampleInput struct {
	Name        string
	Description *string
}

type ExamplePage struct {
	Page     int
	PageSize int
	List     []model.Example
	Total    int64
}

type ExampleService interface {
	List(context.Context, int, int) (ExamplePage, error)
	Get(context.Context, int64) (model.Example, error)
	Create(context.Context, ExampleInput) (model.Example, error)
	Update(context.Context, int64, ExampleInput) (model.Example, error)
	Delete(context.Context, int64) error
}

type exampleService struct{ repository repository.ExampleRepository }

func NewExampleService(repository repository.ExampleRepository) ExampleService {
	return &exampleService{repository: repository}
}

func (s *exampleService) List(ctx context.Context, page, pageSize int) (ExamplePage, error) {
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return ExamplePage{}, ErrValidation
	}
	list, total, err := s.repository.List(ctx, (page-1)*pageSize, pageSize)
	if err != nil {
		return ExamplePage{}, err
	}
	return ExamplePage{Page: page, PageSize: pageSize, List: list, Total: total}, nil
}

func (s *exampleService) Get(ctx context.Context, id int64) (model.Example, error) {
	if id < 1 {
		return model.Example{}, ErrValidation
	}
	return mapNotFound(s.repository.Get(ctx, id))
}

func (s *exampleService) Create(ctx context.Context, input ExampleInput) (model.Example, error) {
	name, description, err := normalize(input)
	if err != nil {
		return model.Example{}, err
	}
	return s.repository.Create(ctx, model.Example{Name: name, Description: description})
}

func (s *exampleService) Update(ctx context.Context, id int64, input ExampleInput) (model.Example, error) {
	if id < 1 {
		return model.Example{}, ErrValidation
	}
	name, description, err := normalize(input)
	if err != nil {
		return model.Example{}, err
	}
	return mapNotFound(s.repository.Update(ctx, model.Example{ID: id, Name: name, Description: description}))
}

func (s *exampleService) Delete(ctx context.Context, id int64) error {
	if id < 1 {
		return ErrValidation
	}
	return mapNotFoundError(s.repository.Delete(ctx, id))
}

func normalize(input ExampleInput) (string, *string, error) {
	name := strings.TrimSpace(input.Name)
	if len(name) == 0 || len([]rune(name)) > 120 {
		return "", nil, ErrValidation
	}
	if input.Description == nil {
		return name, nil, nil
	}
	description := strings.TrimSpace(*input.Description)
	if len([]rune(description)) > 2000 {
		return "", nil, ErrValidation
	}
	if description == "" {
		return name, nil, nil
	}
	return name, &description, nil
}

func mapNotFound(value model.Example, err error) (model.Example, error) {
	return value, mapNotFoundError(err)
}

func mapNotFoundError(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
