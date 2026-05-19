package handler

import (
	"net/http"
	"os"

	"github.com/gen2brain/dlgs"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *Handler) initDBRoutes(api *gin.RouterGroup) {
	group := api.Group("/db")
	{
		group.POST("/select", h.SelectDB)
		group.GET("/info", h.GetInfo)
	}
}

// SelectDBRequest — тело запроса (на будущее, если понадобятся параметры)
type SelectDBRequest struct {
	AllowCreate bool `json:"allow_create,omitempty"` // разрешить создание новой БД
}

// SelectDBResponse — ответ фронтенду
type SelectDBResponse struct {
	Status  string `json:"status"` // "success", "cancelled", "error"
	Path    string `json:"path,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// SelectDB — POST /api/db/select
// Открывает нативный диалог выбора файла и переключает БД
func (h *Handler) SelectDB(c *gin.Context) {
	// 🔹 Парсинг тела запроса (опционально)
	var req SelectDBRequest
	if c.ShouldBindJSON(&req) != nil {
		// Игнорируем ошибку биндинга, если тело пустое — используем дефолты
	}

	h.log.Info("Opening database selection dialog", zap.Bool("allow_create", req.AllowCreate))

	// 🔹 Открываем нативный диалог (блокирующий вызов!)
	// Важно: этот хендлер должен работать в отдельной горутине,
	// если сервер должен обрабатывать другие запросы параллельно
	selectedPath, ok, err := dlgs.File(
		"Выберите файл базы данных",
		"*.db *.sqlite",
		false, // false = выбор файла, true = выбор директории
	)
	if err != nil {
		h.log.Error("File dialog error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, SelectDBResponse{
			Status:  "error",
			Error:   "dialog_failed",
			Message: "Не удалось открыть диалог выбора файла",
		})
		return
	}

	// 🔹 Пользователь отменил выбор
	if !ok {
		h.log.Info("Database selection cancelled by user")
		c.JSON(http.StatusOK, SelectDBResponse{
			Status:  "cancelled",
			Message: "Выбор отменён пользователем",
		})
		return
	}

	// 🔹 Опционально: проверка существования файла
	if _, statErr := os.Stat(selectedPath); os.IsNotExist(statErr) {
		if !req.AllowCreate {
			c.JSON(http.StatusBadRequest, SelectDBResponse{
				Status:  "error",
				Error:   "file_not_found",
				Message: "Файл не существует и создание новой БД не разрешено",
			})
			return
		}
		h.log.Warn("Selected DB file does not exist, will create new", zap.String("path", selectedPath))
	}

	// 🔹 Переключаем БД через интерфейс (потокобезопасно)
	if h.appRef != nil {
		if switchErr := h.appRef.SwitchDatabase(selectedPath); switchErr != nil {
			h.log.Error("Failed to switch database", zap.Error(switchErr), zap.String("path", selectedPath))
			c.JSON(http.StatusInternalServerError, SelectDBResponse{
				Status:  "error",
				Error:   "switch_failed",
				Message: "Не удалось подключиться к выбранной базе данных",
			})
			return
		}
	}

	h.log.Info("Database switched successfully", zap.String("path", selectedPath))

	// 🔹 Успешный ответ
	c.JSON(http.StatusOK, SelectDBResponse{
		Status:  "success",
		Path:    selectedPath,
		Message: "База данных успешно подключена",
	})
}

// GetInfo — GET /api/db/info
// Возвращает информацию о текущей БД
func (h *Handler) GetInfo(c *gin.Context) {
	path := ""
	if h.appRef != nil {
		path = h.appRef.GetDBPath()
	}

	c.JSON(http.StatusOK, gin.H{
		"current_path": path,
		"exists":       path != "" && fileExists(path),
	})
}

// fileExists — вспомогательная функция
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
