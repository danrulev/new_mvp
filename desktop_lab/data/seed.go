package data

import (
	"context"
	"desktop_lab/internal/models"
	"desktop_lab/internal/service"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func SeedData(svc *service.Services, log *zap.Logger) error {
	ctx := context.Background()

	dataDir := filepath.Join("data", "standards")

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
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
		matID, err := getOrCreateMaterial(ctx, svc, payload.Material, log)
		if err != nil {
			return fmt.Errorf("failed to get/create material: %w", err)
		}

		// 2. Регистрируем глобальные измерения и привязываем их к материалу
		dimMap := make(map[string]string)

		// Собираем все уникальные измерения из всех стандартов в файле, чтобы не дублировать
		allDims := make(map[string]DimSeedItem)
		for _, stdJson := range payload.Standards {
			for _, dimJson := range stdJson.Dimensions {
				allDims[dimJson.KeyName] = dimJson
			}
		}

		for keyName, dimJson := range allDims {
			dimID, err := getOrCreateDimension(ctx, svc, dimJson, log)
			if err != nil {
				return fmt.Errorf("failed to get/create dimension %s: %w", keyName, err)
			}
			if dimID == "" {
				return fmt.Errorf("dimension ID is empty for %s", keyName)
			}
			dimMap[keyName] = dimID

			// Привязываем измерение к материалу через сервис
			if err := linkDimensionToMaterial(ctx, svc, matID, dimID, true, log); err != nil {
				return fmt.Errorf("failed to link dimension %s to material: %w", keyName, err)
			}
		}

		// 3. Создаем Стандарты и Методы
		for _, stdJson := range payload.Standards {
			req := models.CreateStandardRequest{
				MaterialID:  matID,
				Name:        stdJson.Name,
				Description: stdJson.Description,
				Dimensions:  make([]models.ContextDimensionDTO, len(stdJson.Dimensions)),
				Methods:     make([]models.CreateMethodDTO, len(stdJson.Methods)),
			}

			// Маппинг Dimensions (передаем key_name, сервис сам найдет ID и создаст связь)
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
					IsMandatory: m.IsMandatory,
					Inputs:      make([]models.MethodInputDTO, len(m.Inputs)),
					Limits:      make([]models.CreateLimitDTO, len(m.Limits)),
				}

				for j, inp := range m.Inputs {
					req.Methods[i].Inputs[j] = models.MethodInputDTO{
						ParamKey:   inp.ParamKey,
						Label:      inp.Label,
						Unit:       inp.Unit,
						InputType:  inp.InputType,
						IsRequired: inp.IsRequired,
					}
				}

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

			// Вызов сервиса создания стандарта
			_, err := svc.Standards.CreateStandard(ctx, req)
			if err != nil {
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

// --- Helper Functions (Только через сервисы) ---

func getOrCreateMaterial(ctx context.Context, svc *service.Services, item MaterialSeedItem, log *zap.Logger) (string, error) {
	// Пытаемся найти существующий
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

	// Создаем новый через сервис
	newMat, err := svc.Materials.Create(ctx, item.Name, item.Code)
	if err != nil {
		return "", err
	}

	log.Debug("Material created", zap.String("name", item.Name), zap.String("id", newMat.ID))
	return newMat.ID, nil
}

func getOrCreateDimension(ctx context.Context, svc *service.Services, dim DimSeedItem, log *zap.Logger) (string, error) {
	// 1. Пытаемся найти через сервис
	existing, err := svc.Dimensions.GetDimensionByKey(ctx, dim.KeyName)
	if err == nil && existing.ID != "" {
		log.Debug("Dimension exists", zap.String("key", dim.KeyName))
		return existing.ID, nil
	}

	// 2. Создаем через сервис
	newDim := models.ContextDimension{
		ID:             uuid.New().String(),
		KeyName:        dim.KeyName,
		Label:          dim.Label,
		DataType:       dim.DataType,
		PossibleValues: dim.PossibleValues,
	}

	err = svc.Dimensions.AddDimension(ctx, newDim)
	if err != nil {
		// Если ошибка уникальности (кто-то создал параллельно), пробуем найти снова
		if strings.Contains(err.Error(), "UNIQUE") {
			existing, err = svc.Dimensions.GetDimensionByKey(ctx, dim.KeyName)
			if err == nil && existing.ID != "" {
				return existing.ID, nil
			}
		}
		return "", fmt.Errorf("failed to add dimension: %w", err)
	}

	// 3. Находим созданное, чтобы получить ID (если AddDimension не вернул его явно)
	existing, err = svc.Dimensions.GetDimensionByKey(ctx, dim.KeyName)
	if err != nil || existing.ID == "" {
		return "", fmt.Errorf("dimension created but not found: %w", err)
	}

	log.Debug("Dimension created", zap.String("key", dim.KeyName), zap.String("id", existing.ID))
	return existing.ID, nil
}

func linkDimensionToMaterial(ctx context.Context, svc *service.Services, matID, dimID string, isRequired bool, log *zap.Logger) error {
	if dimID == "" {
		return fmt.Errorf("cannot link empty dimension ID")
	}

	// Вызываем метод сервиса для связи
	err := svc.Materials.AddContextDimensionToMaterial(ctx, matID, dimID, isRequired)
	if err != nil {
		// Игнорируем ошибку уникальности, если связь уже есть
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil
		}
		// Логируем ошибку FK, если она возникнет (значит измерения нет в БД)
		if strings.Contains(err.Error(), "FOREIGN KEY") {
			return fmt.Errorf("FK constraint failed: dimension %s does not exist", dimID)
		}
		return err
	}
	log.Debug("Dimension linked to material", zap.String("mat", matID), zap.String("dim", dimID))
	return nil
}

// --- Structs ---
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
	Operator      string `json:"operator"`
	ExpectedValue string `json:"expected_value"`
}
