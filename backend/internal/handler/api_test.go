package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/XiaomaiTX/go-react-starter/internal/api/generated"
	"github.com/XiaomaiTX/go-react-starter/internal/model"
	"github.com/XiaomaiTX/go-react-starter/internal/service"
)

type fakeExamples struct {
	page service.ExamplePage
	err  error
}

func (f *fakeExamples) List(context.Context, int, int) (service.ExamplePage, error) {
	return f.page, f.err
}
func (f *fakeExamples) Get(context.Context, int64) (model.Example, error) {
	return model.Example{}, f.err
}
func (f *fakeExamples) Create(context.Context, service.ExampleInput) (model.Example, error) {
	return model.Example{}, f.err
}
func (f *fakeExamples) Update(context.Context, int64, service.ExampleInput) (model.Example, error) {
	return model.Example{}, f.err
}
func (f *fakeExamples) Delete(context.Context, int64) error { return f.err }

func TestReadyHealthRequiresBothDependencies(t *testing.T) {
	api := NewAPI(&fakeExamples{}, func(context.Context) error { return nil }, func(context.Context) error { return errors.New("redis unavailable") })
	response, err := api.GetReadyHealth(context.Background(), generated.GetReadyHealthRequestObject{})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	if err := response.VisitGetReadyHealthResponse(recorder); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("状态码为 %d", recorder.Code)
	}
}

func TestListExamplesWritesValidationProblem(t *testing.T) {
	api := NewAPI(&fakeExamples{err: service.ErrValidation}, func(context.Context) error { return nil }, func(context.Context) error { return nil })
	response, err := api.ListExamples(context.Background(), generated.ListExamplesRequestObject{Params: generated.ListExamplesParams{Page: pointer(int32(0))}})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	if err := response.VisitListExamplesResponse(recorder); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("状态码为 %d", recorder.Code)
	}
}

func pointer[T any](value T) *T { return &value }
