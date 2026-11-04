package handler

import (
	"context"
	"errors"
	"time"

	"github.com/XiaomaiTX/go-react-starter/internal/api/generated"
	"github.com/XiaomaiTX/go-react-starter/internal/model"
	"github.com/XiaomaiTX/go-react-starter/internal/problem"
	"github.com/XiaomaiTX/go-react-starter/internal/service"
)

type Probe func(context.Context) error

type API struct {
	examples  service.ExampleService
	database  Probe
	probeWait time.Duration
}

var _ generated.StrictServerInterface = (*API)(nil)

func NewAPI(examples service.ExampleService, database Probe) *API {
	return &API{examples: examples, database: database, probeWait: 2 * time.Second}
}

func (api *API) ListExamples(ctx context.Context, request generated.ListExamplesRequestObject) (generated.ListExamplesResponseObject, error) {
	page, pageSize := 1, 20
	if request.Params.Page != nil {
		page = int(*request.Params.Page)
	}
	if request.Params.PageSize != nil {
		pageSize = int(*request.Params.PageSize)
	}
	result, err := api.examples.List(ctx, page, pageSize)
	if err != nil {
		return listProblem(err), nil
	}
	return generated.ListExamples200JSONResponse(generated.ExampleList{Page: int32(result.Page), PageSize: int32(result.PageSize), List: examples(result.List), Total: result.Total}), nil
}

func (api *API) CreateExample(ctx context.Context, request generated.CreateExampleRequestObject) (generated.CreateExampleResponseObject, error) {
	if request.Body == nil {
		return createValidation(), nil
	}
	result, err := api.examples.Create(ctx, input(*request.Body))
	if err != nil {
		return createProblem(err), nil
	}
	return generated.CreateExample201JSONResponse(example(result)), nil
}

func (api *API) GetExample(ctx context.Context, request generated.GetExampleRequestObject) (generated.GetExampleResponseObject, error) {
	result, err := api.examples.Get(ctx, request.Id)
	if err != nil {
		return getProblem(err), nil
	}
	return generated.GetExample200JSONResponse(example(result)), nil
}

func (api *API) UpdateExample(ctx context.Context, request generated.UpdateExampleRequestObject) (generated.UpdateExampleResponseObject, error) {
	if request.Body == nil {
		return updateValidation(), nil
	}
	result, err := api.examples.Update(ctx, request.Id, input(*request.Body))
	if err != nil {
		return updateProblem(err), nil
	}
	return generated.UpdateExample200JSONResponse(example(result)), nil
}

func (api *API) DeleteExample(ctx context.Context, request generated.DeleteExampleRequestObject) (generated.DeleteExampleResponseObject, error) {
	if err := api.examples.Delete(ctx, request.Id); err != nil {
		return deleteProblem(err), nil
	}
	return generated.DeleteExample204Response{}, nil
}

func (api *API) GetLiveHealth(_ context.Context, _ generated.GetLiveHealthRequestObject) (generated.GetLiveHealthResponseObject, error) {
	return generated.GetLiveHealth200JSONResponse(generated.HealthStatus{Status: "ok"}), nil
}

func (api *API) GetReadyHealth(ctx context.Context, _ generated.GetReadyHealthRequestObject) (generated.GetReadyHealthResponseObject, error) {
	probeContext, cancel := context.WithTimeout(ctx, api.probeWait)
	defer cancel()
	if err := api.database(probeContext); err != nil {
		return readyInternal(), nil
	}
	return generated.GetReadyHealth200JSONResponse(generated.HealthStatus{Status: "ok"}), nil
}

func input(value generated.ExampleInput) service.ExampleInput {
	return service.ExampleInput{Name: value.Name, Description: value.Description}
}

func examples(values []model.Example) []generated.Example {
	result := make([]generated.Example, 0, len(values))
	for _, value := range values {
		result = append(result, example(value))
	}
	return result
}

func example(value model.Example) generated.Example {
	return generated.Example{Id: value.ID, Name: value.Name, Description: value.Description, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}

func generatedProblem(value problem.Details) generated.Problem {
	return generated.Problem{Type: value.Type, Title: value.Title, Status: value.Status, Detail: value.Detail, Instance: value.Instance}
}

func validationProblem() generated.ValidationProblemApplicationProblemPlusJSONResponse {
	return generated.ValidationProblemApplicationProblemPlusJSONResponse(generatedProblem(problem.Validation("请检查请求参数或内容")))
}

func internalProblem() generated.InternalProblemApplicationProblemPlusJSONResponse {
	return generated.InternalProblemApplicationProblemPlusJSONResponse(generatedProblem(problem.Internal()))
}

func notFoundProblem() generated.NotFoundProblemApplicationProblemPlusJSONResponse {
	return generated.NotFoundProblemApplicationProblemPlusJSONResponse(generatedProblem(problem.NotFound()))
}

func listProblem(err error) generated.ListExamplesResponseObject {
	if errors.Is(err, service.ErrValidation) {
		return generated.ListExamples400ApplicationProblemPlusJSONResponse{ValidationProblemApplicationProblemPlusJSONResponse: validationProblem()}
	}
	return generated.ListExamples500ApplicationProblemPlusJSONResponse{InternalProblemApplicationProblemPlusJSONResponse: internalProblem()}
}
func createValidation() generated.CreateExampleResponseObject {
	return generated.CreateExample400ApplicationProblemPlusJSONResponse{ValidationProblemApplicationProblemPlusJSONResponse: validationProblem()}
}
func createProblem(err error) generated.CreateExampleResponseObject {
	if errors.Is(err, service.ErrValidation) {
		return createValidation()
	}
	return generated.CreateExample500ApplicationProblemPlusJSONResponse{InternalProblemApplicationProblemPlusJSONResponse: internalProblem()}
}
func getProblem(err error) generated.GetExampleResponseObject {
	if errors.Is(err, service.ErrValidation) {
		return generated.GetExample400ApplicationProblemPlusJSONResponse{ValidationProblemApplicationProblemPlusJSONResponse: validationProblem()}
	}
	if errors.Is(err, service.ErrNotFound) {
		return generated.GetExample404ApplicationProblemPlusJSONResponse{NotFoundProblemApplicationProblemPlusJSONResponse: notFoundProblem()}
	}
	return generated.GetExample500ApplicationProblemPlusJSONResponse{InternalProblemApplicationProblemPlusJSONResponse: internalProblem()}
}
func updateValidation() generated.UpdateExampleResponseObject {
	return generated.UpdateExample400ApplicationProblemPlusJSONResponse{ValidationProblemApplicationProblemPlusJSONResponse: validationProblem()}
}
func updateProblem(err error) generated.UpdateExampleResponseObject {
	if errors.Is(err, service.ErrValidation) {
		return updateValidation()
	}
	if errors.Is(err, service.ErrNotFound) {
		return generated.UpdateExample404ApplicationProblemPlusJSONResponse{NotFoundProblemApplicationProblemPlusJSONResponse: notFoundProblem()}
	}
	return generated.UpdateExample500ApplicationProblemPlusJSONResponse{InternalProblemApplicationProblemPlusJSONResponse: internalProblem()}
}
func deleteProblem(err error) generated.DeleteExampleResponseObject {
	if errors.Is(err, service.ErrValidation) {
		return generated.DeleteExample400ApplicationProblemPlusJSONResponse{ValidationProblemApplicationProblemPlusJSONResponse: validationProblem()}
	}
	if errors.Is(err, service.ErrNotFound) {
		return generated.DeleteExample404ApplicationProblemPlusJSONResponse{NotFoundProblemApplicationProblemPlusJSONResponse: notFoundProblem()}
	}
	return generated.DeleteExample500ApplicationProblemPlusJSONResponse{InternalProblemApplicationProblemPlusJSONResponse: internalProblem()}
}
func readyInternal() generated.GetReadyHealthResponseObject {
	return generated.GetReadyHealth500ApplicationProblemPlusJSONResponse{InternalProblemApplicationProblemPlusJSONResponse: internalProblem()}
}
