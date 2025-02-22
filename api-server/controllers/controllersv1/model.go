package controllersv1

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huandu/xstrings"
	"github.com/pkg/errors"

	"github.com/bentoml/yatai/api-server/models"
	"github.com/bentoml/yatai/api-server/services"
	"github.com/bentoml/yatai/api-server/transformers/transformersv1"
	"github.com/bentoml/yatai/common/utils"
	"github.com/bentoml/yatai/schemas/modelschemas"
	"github.com/bentoml/yatai/schemas/schemasv1"
)

type modelController struct {
	baseController
}

var ModelController = modelController{}

type GetModelSchema struct {
	GetModelRepositorySchema
	Version string `path:"version"`
}

func (s *GetModelSchema) GetModel(ctx context.Context) (*models.Model, error) {
	modelRepository, err := s.GetModelRepository(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "get modelRepository %s", modelRepository.Name)
	}
	model, err := services.ModelService.GetByVersion(ctx, modelRepository.ID, s.Version)
	if err != nil {
		return nil, errors.Wrapf(err, "get modelRepository %s model %s", modelRepository.Name, s.Version)
	}
	return model, nil
}

func (c *modelController) canView(ctx context.Context, model *models.Model) error {
	modelRepository, err := services.ModelRepositoryService.GetAssociatedModelRepository(ctx, model)
	if err != nil {
		return errors.Wrap(err, "get associated modelRepository")
	}
	return ModelRepositoryController.canView(ctx, modelRepository)
}

func (c *modelController) canUpdate(ctx context.Context, model *models.Model) error {
	modelRepository, err := services.ModelRepositoryService.GetAssociatedModelRepository(ctx, model)
	if err != nil {
		return errors.Wrap(err, "get associated modelRepository")
	}
	return ModelRepositoryController.canUpdate(ctx, modelRepository)
}

func (c *modelController) canOperate(ctx context.Context, model *models.Model) error {
	modelRepository, err := services.ModelRepositoryService.GetAssociatedModelRepository(ctx, model)
	if err != nil {
		return errors.Wrap(err, "get associated modelRepository")
	}
	return ModelRepositoryController.canOperate(ctx, modelRepository)
}

type CreateModelSchema struct {
	schemasv1.CreateModelSchema
	GetModelRepositorySchema
}

func (c *modelController) Create(ctx *gin.Context, schema *CreateModelSchema) (*schemasv1.ModelSchema, error) {
	user, err := services.GetCurrentUser(ctx)
	if err != nil {
		return nil, err
	}
	modelRepository, err := schema.GetModelRepository(ctx)
	if err != nil {
		return nil, err
	}
	if err = ModelRepositoryController.canUpdate(ctx, modelRepository); err != nil {
		return nil, err
	}
	buildAt, err := time.Parse("2006-01-02 15:04:05.000000", schema.BuildAt)
	if err != nil {
		return nil, errors.Wrap(err, "parse build at")
	}
	model, err := services.ModelService.Create(ctx, services.CreateModelOption{
		CreatorId:         user.ID,
		ModelRepositoryId: modelRepository.ID,
		Version:           schema.Version,
		Description:       schema.Description,
		Manifest:          schema.Manifest,
		BuildAt:           buildAt,
		Labels:            schema.Labels,
	})
	if err != nil {
		return nil, errors.Wrap(err, "create modelRepository model")
	}
	return transformersv1.ToModelSchema(ctx, model)
}

func (c *modelController) PreSignUploadUrl(ctx *gin.Context, schema *GetModelSchema) (*schemasv1.ModelSchema, error) {
	model, err := schema.GetModel(ctx)
	if err != nil {
		return nil, err
	}
	if err = c.canUpdate(ctx, model); err != nil {
		return nil, err
	}
	url, err := services.ModelService.PreSignUploadUrl(ctx, model)
	if err != nil {
		return nil, errors.Wrap(err, "pre sign s3 upload url")
	}
	modelSchema, err := transformersv1.ToModelSchema(ctx, model)
	if err != nil {
		return nil, err
	}
	modelSchema.PresignedUploadUrl = url.String()
	return modelSchema, nil
}

func (c *modelController) PreSignDownloadUrl(ctx *gin.Context, schema *GetModelSchema) (*schemasv1.ModelSchema, error) {
	model, err := schema.GetModel(ctx)
	if err != nil {
		return nil, err
	}
	if err = c.canUpdate(ctx, model); err != nil {
		return nil, err
	}
	url, err := services.ModelService.PreSignDownloadUrl(ctx, model)
	if err != nil {
		return nil, errors.Wrap(err, "pre sign s3 download url")
	}
	modelSchema, err := transformersv1.ToModelSchema(ctx, model)
	if err != nil {
		return nil, err
	}
	modelSchema.PresignedDownloadUrl = url.String()
	return modelSchema, nil
}

type UpdateModelSchema struct {
	schemasv1.UpdateModelSchema
	GetModelSchema
}

func (c *modelController) Update(ctx *gin.Context, schema *UpdateModelSchema) (*schemasv1.ModelSchema, error) {
	model, err := schema.GetModel(ctx)
	if err != nil {
		return nil, err
	}
	if err = c.canUpdate(ctx, model); err != nil {
		return nil, err
	}
	model, err = services.ModelService.Update(ctx, model, services.UpdateModelOption{
		Labels: schema.Labels,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Update model")
	}
	return transformersv1.ToModelSchema(ctx, model)
}

func (c *modelController) StartUpload(ctx *gin.Context, schema *GetModelSchema) (*schemasv1.ModelSchema, error) {
	model, err := schema.GetModel(ctx)
	if err != nil {
		return nil, err
	}
	if err = c.canUpdate(ctx, model); err != nil {
		return nil, err
	}
	uploadStatus := modelschemas.ModelUploadStatusUploading
	now := time.Now()
	nowPtr := &now
	model, err = services.ModelService.Update(ctx, model, services.UpdateModelOption{
		UploadStatus:    &uploadStatus,
		UploadStartedAt: &nowPtr,
	})
	if err != nil {
		return nil, errors.Wrap(err, "update model")
	}
	return transformersv1.ToModelSchema(ctx, model)
}

type FinishUploadModelSchema struct {
	schemasv1.FinishUploadModelSchema
	GetModelSchema
}

func (c *modelController) FinishUpload(ctx *gin.Context, schema *FinishUploadModelSchema) (*schemasv1.ModelSchema, error) {
	model, err := schema.GetModel(ctx)
	if err != nil {
		return nil, err
	}
	if err = c.canUpdate(ctx, model); err != nil {
		return nil, err
	}
	now := time.Now()
	nowPtr := &now
	model, err = services.ModelService.Update(ctx, model, services.UpdateModelOption{
		UploadStatus:         schema.Status,
		UploadFinishedAt:     &nowPtr,
		UploadFinishedReason: schema.Reason,
	})
	if err != nil {
		return nil, errors.Wrap(err, "update model")
	}
	if schema.Status != nil {
		user, err := services.GetCurrentUser(ctx)
		if err != nil {
			return nil, err
		}
		modelRepository, err := services.ModelRepositoryService.GetAssociatedModelRepository(ctx, model)
		if err != nil {
			return nil, err
		}
		org, err := services.OrganizationService.GetAssociatedOrganization(ctx, modelRepository)
		if err != nil {
			return nil, err
		}
		createEventOpt := services.CreateEventOption{
			CreatorId:      user.ID,
			OrganizationId: &org.ID,
			ResourceType:   modelschemas.ResourceTypeModel,
			ResourceId:     model.ID,
			Status:         modelschemas.EventStatusSuccess,
			OperationName:  "pushed",
		}
		if *schema.Status != modelschemas.ModelUploadStatusSuccess {
			createEventOpt.Status = modelschemas.EventStatusFailed
		}
		if _, err = services.EventService.Create(ctx, createEventOpt); err != nil {
			return nil, errors.Wrap(err, "create event")
		}
	}
	return transformersv1.ToModelSchema(ctx, model)
}

func (c *modelController) RecreateImageBuilderJob(ctx *gin.Context, schema *GetModelSchema) (*schemasv1.ModelSchema, error) {
	model, err := schema.GetModel(ctx)
	if err != nil {
		return nil, err
	}
	if err = c.canUpdate(ctx, model); err != nil {
		return nil, err
	}
	model, err = services.ModelService.CreateImageBuilderJob(ctx, model)
	if err != nil {
		return nil, err
	}
	return transformersv1.ToModelSchema(ctx, model)
}

func (c *modelController) ListImageBuilderPods(ctx *gin.Context, schema *GetModelSchema) ([]*schemasv1.KubePodSchema, error) {
	model, err := schema.GetModel(ctx)
	if err != nil {
		return nil, err
	}
	if err = c.canView(ctx, model); err != nil {
		return nil, err
	}
	pods, err := services.ModelService.ListImageBuilderPods(ctx, model)
	if err != nil {
		return nil, err
	}
	return transformersv1.ToKubePodSchemas(ctx, pods)
}

func (c *modelController) Get(ctx *gin.Context, schema *GetModelSchema) (*schemasv1.ModelFullSchema, error) {
	model, err := schema.GetModel(ctx)
	if err != nil {
		return nil, err
	}
	if err = c.canView(ctx, model); err != nil {
		return nil, err
	}
	return transformersv1.ToModelFullSchema(ctx, model)
}

type ListModelDeploymentSchema struct {
	schemasv1.ListQuerySchema
	GetModelSchema
}

func (c *modelController) ListDeployment(ctx *gin.Context, schema *ListModelDeploymentSchema) (*schemasv1.DeploymentListSchema, error) {
	model, err := schema.GetModel(ctx)
	if err != nil {
		return nil, err
	}
	if err = c.canView(ctx, model); err != nil {
		return nil, err
	}
	bentos, _, err := services.BentoService.List(ctx, services.ListBentoOption{
		ModelIds: &[]uint{model.ID},
	})
	if err != nil {
		return nil, err
	}
	bentoIds := make([]uint, 0, len(bentos))
	for _, bento := range bentos {
		bentoIds = append(bentoIds, bento.ID)
	}
	deployments, total, err := services.DeploymentService.List(ctx, services.ListDeploymentOption{
		BaseListOption: services.BaseListOption{
			Start:  &schema.Start,
			Count:  &schema.Count,
			Search: schema.Search,
		},
		BentoIds: &bentoIds,
	})
	if err != nil {
		return nil, err
	}
	deploymentSchemas, err := transformersv1.ToDeploymentSchemas(ctx, deployments)
	if err != nil {
		return nil, err
	}
	return &schemasv1.DeploymentListSchema{
		BaseListSchema: schemasv1.BaseListSchema{
			Total: total,
			Start: schema.Start,
			Count: schema.Count,
		},
		Items: deploymentSchemas,
	}, nil
}

type ListModelBentoSchema struct {
	schemasv1.ListQuerySchema
	GetModelSchema
}

func (c *modelController) ListBento(ctx *gin.Context, schema *ListModelBentoSchema) (*schemasv1.BentoWithRepositoryListSchema, error) {
	model, err := schema.GetModel(ctx)
	if err != nil {
		return nil, err
	}
	if err = c.canView(ctx, model); err != nil {
		return nil, err
	}
	bentos, total, err := services.BentoService.List(ctx, services.ListBentoOption{
		BaseListOption: services.BaseListOption{
			Start:  &schema.Start,
			Count:  &schema.Count,
			Search: schema.Search,
		},
		ModelIds: &[]uint{model.ID},
	})
	if err != nil {
		return nil, err
	}
	bentoSchemas, err := transformersv1.ToBentoWithRepositorySchemas(ctx, bentos)
	if err != nil {
		return nil, err
	}
	return &schemasv1.BentoWithRepositoryListSchema{
		BaseListSchema: schemasv1.BaseListSchema{
			Total: total,
			Start: schema.Start,
			Count: schema.Count,
		},
		Items: bentoSchemas,
	}, nil
}

type ListModelSchema struct {
	schemasv1.ListQuerySchema
	GetModelRepositorySchema
}

func (c *modelController) List(ctx *gin.Context, schema *ListModelSchema) (*schemasv1.ModelListSchema, error) {
	modelRepository, err := schema.GetModelRepository(ctx)
	if err != nil {
		return nil, err
	}
	if err = ModelRepositoryController.canView(ctx, modelRepository); err != nil {
		return nil, err
	}

	models_, total, err := services.ModelService.List(ctx, services.ListModelOption{
		BaseListOption: services.BaseListOption{
			Start:  utils.UintPtr(schema.Start),
			Count:  utils.UintPtr(schema.Count),
			Search: schema.Search,
		},
		ModelRepositoryId: utils.UintPtr(modelRepository.ID),
	})
	if err != nil {
		return nil, errors.Wrap(err, "list models")
	}

	modelSchemas, err := transformersv1.ToModelSchemas(ctx, models_)
	return &schemasv1.ModelListSchema{
		BaseListSchema: schemasv1.BaseListSchema{
			Total: total,
			Start: schema.Start,
			Count: schema.Count,
		},
		Items: modelSchemas,
	}, err
}

type ListAllModelSchema struct {
	schemasv1.ListQuerySchema
	GetOrganizationSchema
}

func (c *modelController) ListAll(ctx *gin.Context, schema *ListAllModelSchema) (*schemasv1.ModelWithRepositoryListSchema, error) {
	organization, err := schema.GetOrganization(ctx)
	if err != nil {
		return nil, err
	}

	if err = OrganizationController.canView(ctx, organization); err != nil {
		return nil, err
	}

	listOpt := services.ListModelOption{
		BaseListOption: services.BaseListOption{
			Start:  utils.UintPtr(schema.Start),
			Count:  utils.UintPtr(schema.Count),
			Search: schema.Search,
		},
		OrganizationId: utils.UintPtr(organization.ID),
	}

	queryMap := schema.Q.ToMap()
	for k, v := range queryMap {
		if k == schemasv1.KeyQIn {
			fieldNames := make([]string, 0, len(v.([]string)))
			for _, fieldName := range v.([]string) {
				if _, ok := map[string]struct{}{
					"name":        {},
					"description": {},
				}[fieldName]; !ok {
					continue
				}
				fieldNames = append(fieldNames, fieldName)
			}
			listOpt.KeywordFieldNames = &fieldNames
		}
		if k == schemasv1.KeyQKeywords {
			listOpt.Keywords = utils.StringSlicePtr(v.([]string))
		}
		if k == "module" {
			listOpt.Modules = utils.StringSlicePtr(v.([]string))
		}
		if k == "creator" {
			userNames, err := processUserNamesFromQ(ctx, v.([]string))
			if err != nil {
				return nil, err
			}
			users, err := services.UserService.ListByNames(ctx, userNames)
			if err != nil {
				return nil, err
			}
			userIds := make([]uint, 0, len(users))
			for _, user := range users {
				userIds = append(userIds, user.ID)
			}
			listOpt.CreatorIds = utils.UintSlicePtr(userIds)
		}
		if k == "sort" {
			fieldName, _, order := xstrings.LastPartition(v.([]string)[0], "-")
			if _, ok := map[string]struct{}{
				"created_at": {},
				"build_at":   {},
				"size":       {},
			}[fieldName]; !ok {
				continue
			}
			if fieldName == "size" {
				fieldName = "manifest->'size_bytes'"
			}
			if _, ok := map[string]struct{}{
				"desc": {},
				"asc":  {},
			}[order]; !ok {
				continue
			}
			listOpt.Order = utils.StringPtr(fmt.Sprintf("model.%s %s", fieldName, strings.ToUpper(order)))
		}
		if k == "label" {
			labelsSchema := services.ParseQueryLabelsToLabelsList(v.([]string))
			listOpt.LabelsList = &labelsSchema
		}
		if k == "-label" {
			labelsSchema := services.ParseQueryLabelsToLabelsList(v.([]string))
			listOpt.LackLabelsList = &labelsSchema
		}
	}
	models_, total, err := services.ModelService.List(ctx, listOpt)
	if err != nil {
		return nil, errors.Wrap(err, "list models")
	}

	modelSchemas, err := transformersv1.ToModelWithRepositorySchemas(ctx, models_)
	return &schemasv1.ModelWithRepositoryListSchema{
		BaseListSchema: schemasv1.BaseListSchema{
			Total: total,
			Start: schema.Start,
			Count: schema.Count,
		},
		Items: modelSchemas,
	}, err
}
