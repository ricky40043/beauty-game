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
		Disabled   bool   `json:"disabled"`
	}

	// 後台要看得到被停用的題目才能重新啟用，所以這裡不能用 GetBank
	// （它已經把停用的濾掉了），改成列出全部再標記狀態。
	out := make([]item, 0, 120)
	for _, mode := range []models.GameMode{models.ModeGroup, models.ModeSolo} {
		all := append(services.AllQuestions(mode), h.questionStore().Custom(mode)...)
		for _, q := range all {
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
				Disabled:   h.questionStore().IsDisabled(q.ID),
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

// ─── 題目管理 ────────────────────────────────────────────
//
// 題目原本全部寫死在 Go 程式裡，只有重新編譯才能改。
// 這裡讓後台可以新增自己的題目，以及停用不想出現的內建題。

type questionPayload struct {
	Text       string          `json:"text"`
	Mode       models.GameMode `json:"mode"`
	Category   string          `json:"category"`
	Difficulty int             `json:"difficulty"`
}

// questionStore 後台自訂題目的儲存（跟示範圖的 h.store 是兩回事）
func (h *ExampleHandler) questionStore() *services.QuestionStore {
	return h.questions.Store()
}

// CreateQuestion POST /api/admin/questions
func (h *ExampleHandler) CreateQuestion(c *gin.Context) {
	var req questionPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "題目格式錯誤"})
		return
	}

	question, err := h.questionStore().Add(req.Text, req.Mode, req.Category, req.Difficulty)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"question": question})
}

// UpdateQuestion PUT /api/admin/questions/:questionId
func (h *ExampleHandler) UpdateQuestion(c *gin.Context) {
	id, ok := questionID(c)
	if !ok {
		return
	}

	var req questionPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "題目格式錯誤"})
		return
	}

	question, err := h.questionStore().Update(id, req.Text, req.Category, req.Difficulty)
	if err != nil {
		c.JSON(questionErrorStatus(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"question": question})
}

// DeleteQuestion DELETE /api/admin/questions/:questionId
// 一併把這題的自訂示範圖清掉，避免編號被回收後接到別人的舊圖
func (h *ExampleHandler) DeleteQuestion(c *gin.Context) {
	id, ok := questionID(c)
	if !ok {
		return
	}

	if err := h.questionStore().Delete(id); err != nil {
		c.JSON(questionErrorStatus(err), gin.H{"error": err.Error()})
		return
	}

	h.store.Delete(id)
	c.JSON(http.StatusOK, gin.H{"questionId": id})
}

// SetQuestionDisabled POST /api/admin/questions/:questionId/disabled
// 內建題不能刪，但可以停用讓它不再被抽到
func (h *ExampleHandler) SetQuestionDisabled(c *gin.Context) {
	id, ok := questionID(c)
	if !ok {
		return
	}

	var req struct {
		Disabled bool `json:"disabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "參數格式錯誤"})
		return
	}

	if err := h.questionStore().SetDisabled(id, req.Disabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"questionId": id, "disabled": req.Disabled})
}

func questionErrorStatus(err error) int {
	switch {
	case errors.Is(err, services.ErrQuestionNotFound):
		return http.StatusNotFound
	case errors.Is(err, services.ErrBuiltinNotEditable):
		return http.StatusForbidden
	default:
		return http.StatusBadRequest
	}
}
