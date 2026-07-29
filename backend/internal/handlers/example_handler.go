package handlers

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"beauty-game/internal/examples"
	"beauty-game/internal/models"
	"beauty-game/internal/services"

	"github.com/gin-gonic/gin"
)

// ExampleHandler 題目示範圖：讀取（內建 SVG 或房主上傳）、後台上傳與還原
type ExampleHandler struct {
	store      *examples.Store
	questions  *services.QuestionService
	adminToken string
}

// NewExampleHandler 建立示範圖處理器。
// adminToken 為空字串時不驗證（本機開發用），部署時請設 ADMIN_TOKEN。
func NewExampleHandler(store *examples.Store, questions *services.QuestionService, adminToken string) *ExampleHandler {
	return &ExampleHandler{store: store, questions: questions, adminToken: adminToken}
}

// RequireAdmin 後台寫入操作的驗證中介層
func (h *ExampleHandler) RequireAdmin(c *gin.Context) {
	if h.adminToken == "" {
		c.Next()
		return
	}

	token := c.GetHeader("X-Admin-Token")
	if token == "" {
		token = c.Query("token")
	}

	if token != h.adminToken {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "後台密碼不正確"})
		return
	}

	c.Next()
}

func questionID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("questionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "題號格式錯誤"})
		return 0, false
	}
	return id, true
}

// Get GET /api/questions/:questionId/example
// 房主上傳的版本優先，沒有就退回內建的火柴人示範圖，都沒有回 404。
func (h *ExampleHandler) Get(c *gin.Context) {
	id, ok := questionID(c)
	if !ok {
		return
	}

	if data, contentType, found := h.store.Get(id); found {
		c.Header("Cache-Control", "no-cache")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Example-Source", "custom")
		c.Data(http.StatusOK, contentType, data)
		return
	}

	if svg, found := examples.Builtin(id); found {
		// 快取時間刻意壓短：房主把自訂圖丟進資料夾後，
		// 不用清快取或重開服務，最多一分鐘就會換成新圖。
		c.Header("Cache-Control", "public, max-age=60")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Example-Source", "builtin")
		c.Data(http.StatusOK, "image/svg+xml", []byte(svg))
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "這題還沒有示範圖"})
}

// List GET /api/admin/examples
// 後台用：列出所有題目與它們目前的示範圖狀態
func (h *ExampleHandler) List(c *gin.Context) {
	type item struct {
		models.Question
		HasBuiltin bool   `json:"hasBuiltin"`
		HasCustom  bool   `json:"hasCustom"`
		ImageURL   string `json:"imageUrl"`
	}

	out := make([]item, 0, 80)
	for _, mode := range []models.GameMode{models.ModeGroup, models.ModeSolo} {
		for _, q := range h.questions.GetBank(mode) {
			hasCustom := h.store.Has(q.ID)
			hasBuiltin := examples.HasBuiltin(q.ID)

			url := ""
			if hasCustom || hasBuiltin {
				url = "/api/questions/" + strconv.Itoa(q.ID) + "/example"
			}

			out = append(out, item{
				Question:   q,
				HasBuiltin: hasBuiltin,
				HasCustom:  hasCustom,
				ImageURL:   url,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"questions": out,
		"storage":   h.store.Dir(),
		"protected": h.adminToken != "",
	})
}

// Upload POST /api/admin/questions/:questionId/example
func (h *ExampleHandler) Upload(c *gin.Context) {
	id, ok := questionID(c)
	if !ok {
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, examples.MaxOverrideBytes+1024)

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "沒有收到圖片檔案"})
		return
	}

	opened, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "讀取圖片失敗"})
		return
	}
	defer opened.Close()

	data, err := io.ReadAll(io.LimitReader(opened, examples.MaxOverrideBytes))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "讀取圖片失敗"})
		return
	}

	if err := h.store.Put(id, data); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, examples.ErrTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"questionId": id,
		"imageUrl":   "/api/questions/" + strconv.Itoa(id) + "/example",
		"source":     "custom",
	})
}

// Delete DELETE /api/admin/questions/:questionId/example
// 只刪掉房主上傳的版本，內建示範圖會自動接手
func (h *ExampleHandler) Delete(c *gin.Context) {
	id, ok := questionID(c)
	if !ok {
		return
	}

	h.store.Delete(id)
	c.JSON(http.StatusOK, gin.H{
		"questionId":      id,
		"revertedToBuilt": examples.HasBuiltin(id),
	})
}
