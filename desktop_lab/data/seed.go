package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"desktop_lab/internal/models"
	"desktop_lab/internal/service"

	"go.uber.org/zap"
)

// SeedData загружает справочники из JSON файлов в базу данных
func SeedData(svc *service.Services, log *zap.Logger) error {
	ctx := context.Background()

	// Путь к папке с данными (относительно корня проекта или exe)
	// При разработке: ./data/standards
	// При продакшене: нужно использовать embed или путь рядом с exe
	dataDir := filepath.Join("data", "standards")

	// Проверка существования папки
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		// Попытка найти относительно exe (для сбилженного приложения)
		execPath, _ := os.Executable()
		execDir := filepath.Dir(execPath)
		dataDir = filepath.Join(execDir, "data", "standards")

		if _, err := os.Stat(dataDir); os.IsNotExist(err) {
			log.Warn("Standards data directory not found, skipping seed")
			return nil
		}
	}

	log.Info("Starting data seeding...", zap.String("path", dataDir))

	count := 0
	err := filepath.WalkDir(dataDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}

		log.Debug("Processing file", zap.String("file", d.Name()))

		fileData, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", d.Name(), err)
		}

		var payload StandardSeedPayload
		if err := json.Unmarshal(fileData, &payload); err != nil {
			return fmt.Errorf("failed to parse JSON %s: %w", d.Name(), err)
		}

		// 1. Создаем или находим Материал
		matID, err := svc.Materials.GetOrCreate(ctx, payload.Material.Name, payload.Material.Code)
		if err != nil {
			return fmt.Errorf("failed to get/create material: %w", err)
		}

		// 2. Создаем Стандарты и Методы
		for _, stdJson := range payload.Standards {
			req := models.CreateStandardRequest{
				MaterialID:  matID,
				Name:        stdJson.Name,
				Description: stdJson.Description,
				Dimensions:  make([]models.ContextDimensionDTO, len(stdJson.Dimensions)),
				Methods:     make([]models.CreateMethodDTO, len(stdJson.Methods)),
			}

			// Маппинг Dimensions
			for i, dim := range stdJson.Dimensions {
				req.Dimensions[i] = models.ContextDimensionDTO{
					KeyName:        dim.KeyName,
					Label:          dim.Label,
					DataType:       dim.DataType,
					PossibleValues: dim.PossibleValues,
				}
			}

			// Маппинг Methods
			for i, m := range stdJson.Methods {
				req.Methods[i] = models.CreateMethodDTO{
					Code:        m.Code,
					Name:        m.Name,
					Unit:        m.Unit,
					FormulaExpr: m.FormulaExpr,
					Inputs:      make([]models.MethodInputDTO, len(m.Inputs)),
					Limits:      make([]models.CreateLimitDTO, len(m.Limits)),
				}

				// Inputs
				for j, inp := range m.Inputs {
					req.Methods[i].Inputs[j] = models.MethodInputDTO{
						ParamKey:   inp.ParamKey,
						Label:      inp.Label,
						Unit:       inp.Unit,
						InputType:  inp.InputType,
						IsRequired: inp.IsRequired,
					}
				}

				// Limits
				for k, lim := range m.Limits {
					req.Methods[i].Limits[k] = models.CreateLimitDTO{
						LimitType:  lim.LimitType,
						MinValue:   lim.MinValue,
						MaxValue:   lim.MaxValue,
						Conditions: make([]models.ConditionDTO, len(lim.Conditions)),
					}

					for l, cond := range lim.Conditions {
						req.Methods[i].Limits[k].Conditions[l] = models.ConditionDTO{
							DimensionKey:  cond.DimensionKey,
							Operator:      cond.Operator,
							ExpectedValue: cond.ExpectedValue,
						}
					}
				}
			}

			// Вызов сервиса создания
			_, err := svc.Standards.CreateStandard(ctx, req)
			if err != nil {
				// Игнорируем ошибку уникальности, если стандарт уже есть
				if strings.Contains(err.Error(), "UNIQUE constraint failed") {
					log.Debug("Standard already exists", zap.String("name", stdJson.Name))
					continue
				}
				return fmt.Errorf("failed to create standard %s: %w", stdJson.Name, err)
			}
			log.Info("Standard created", zap.String("name", stdJson.Name))
		}

		count++
		return nil
	})

	if err != nil {
		return err
	}

	log.Info("Seeding completed", zap.Int("files_processed", count))
	return nil
}

// --- Вспомогательные структуры для JSON ---

type StandardSeedPayload struct {
	Material  MaterialSeedItem   `json:"material"`
	Standards []StandardSeedItem `json:"standards"`
}

type MaterialSeedItem struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type StandardSeedItem struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Dimensions  []DimSeedItem    `json:"dimensions"`
	Methods     []MethodSeedItem `json:"methods"`
}

type DimSeedItem struct {
	KeyName        string   `json:"key_name"`
	Label          string   `json:"label"`
	DataType       string   `json:"data_type"`
	PossibleValues []string `json:"possible_values"`
}

type MethodSeedItem struct {
	Code        string          `json:"code"`
	Name        string          `json:"name"`
	Unit        string          `json:"unit"`
	FormulaExpr string          `json:"formula_expr"`
	IsMandatory bool            `json:"is_mandatory"`
	Inputs      []InputSeedItem `json:"inputs"`
	Limits      []LimitSeedItem `json:"limits"`
}

type InputSeedItem struct {
	ParamKey   string `json:"param_key"`
	Label      string `json:"label"`
	Unit       string `json:"unit"`
	InputType  string `json:"input_type"`
	IsRequired bool   `json:"is_required"`
}

type LimitSeedItem struct {
	LimitType  string              `json:"limit_type"`
	MinValue   *float64            `json:"min_value"`
	MaxValue   *float64            `json:"max_value"`
	Priority   int                 `json:"priority"`
	Conditions []ConditionSeedItem `json:"conditions"`
}

type ConditionSeedItem struct {
	DimensionKey  string `json:"dimension_key"`
	Operator      string `json:"operator"` // "=", "IN", "!="
	ExpectedValue string `json:"expected_value"`
}

// --- Helper Functions ---

func getOrCreateMaterial(ctx context.Context, svc *service.Services, item MaterialSeedItem, log *zap.Logger) (string, error) {
	// Сначала пробуем найти (если бы был метод GetByName, но его нет в интерфейсе, поэтому создаем с обработкой ошибки)
	// Или просто создаем, ловя ошибку уникальности

	// Генерируем детерминированный ID на основе имени, чтобы не дублировать при перезапуске
	// Это простой хак, лучше делать SELECT сначала

	mat, err := svc.Materials.Create(ctx, item.Name, item.Code)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			// Материал уже есть, нужно найти его ID
			// В текущем интерфейсе Service нет метода GetAll с фильтром,
			// поэтому придется получить все и найти нужный (для сида это ок)
			allMats, err := svc.Materials.GetAll(ctx)
			if err != nil {
				return "", err
			}
			for _, m := range allMats {
				if m.Name == item.Name {
					log.Debug("Material exists", zap.String("name", item.Name))
					return m.ID, nil
				}
			}
		}
		return "", err
	}
	return mat.ID, nil
}
