package tests

// import (
// 	"context"
// 	"desktop_lab/internal/models"
// 	"desktop_lab/internal/service"

// 	"go.uber.org/zap"
// )

// func StartTests(log *zap.Logger, svs *service.Services) {

// }

// func testMaterials(log *zap.Logger, svs *service.Services) {
// 	mat, err := svs.Materials.Create(context.TODO(), "test", "test")
// 	if err != nil {
// 		log.Error("failed to create material", zap.Error(err))
// 		return
// 	} else {
// 		log.Info("material created", zap.Any("mat", mat))
// 	}

// 	mats, err := svs.Materials.GetAll(context.TODO())
// 	if err != nil {
// 		log.Error("failed to get materials", zap.Error(err))
// 		return
// 	} else {
// 		log.Info("materials", zap.Any("mats", mats))
// 	}

// 	mat, err = svs.Materials.GetByID(context.TODO(), mat.ID)
// 	if err != nil {
// 		log.Error("failed to get material", zap.Error(err))
// 		return
// 	} else {
// 		log.Info("material", zap.Any("mat", mat))
// 	}

// 	matID, err := svs.Materials.GetOrCreate(context.TODO(), "test", "test")
// 	if err != nil {
// 		log.Error("failed to get or create material", zap.Error(err))
// 		return
// 	} else {
// 		log.Info("material", zap.Any("matID", matID), zap.Any("matID", mat.ID))
// 	}
// }

// func testStandards(log *zap.Logger, svs *service.Services) {
// 	stdID, err := svs.Standards.CreateStandard(context.TODO(), models.CreateStandardRequest{
// 		MaterialID:  "test",
// 		Name:        "test",
// 		Description: "test",
// 		Dimensions: []models.ContextDimensionDTO{
// 			{
// 				KeyName:        "test",
// 				Label:          "test",
// 				PossibleValues: []string{"test"},
// 				DataType:       "test",
// 			},
// 		},
// 		Methods: []models.CreateMethodDTO{
// 			{
// 				Code:        "test",
// 				Name:        "test",
// 				Unit:        "test",
// 				FormulaExpr: "test",
// 				Inputs: []models.MethodInputDTO{
// 					{
// 						InputType:  "test",
// 						IsRequired: true,
// 						Label:      "test",
// 						ParamKey:   "test",
// 						Unit:       "test",
// 					},
// 				},
// 			},
// 		},
// 	})
// }
