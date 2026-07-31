package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"beauty-game/internal/models"
)

// CustomQuestionIDBase 自訂題目的編號從這裡開始遞增。
//
// 刻意跟其他號段錯開：內建單人題 1001~1508、內建團體題 2001~2025、
// 房主在建房時臨時打的題目 9000+、試玩題 9999。用 30000 起算不會相撞，
// 而且看到編號就知道是哪一類。
const CustomQuestionIDBase = 30000

var (
	// ErrQuestionNotFound 找不到這個自訂題目
	ErrQuestionNotFound = errors.New("找不到這個自訂題目")
	// ErrQuestionTextRequired 題目內容是空的
	ErrQuestionTextRequired = errors.New("題目內容不能是空的")
	// ErrBuiltinNotEditable 內建題目不能改也不能刪
	ErrBuiltinNotEditable = errors.New("內建題目不能編輯或刪除，只能停用")
)

// questionFile 落地的 JSON 結構
type questionFile struct {
	CustomQuestions    []models.Question `json:"customQuestions"`
	DisabledBuiltinIDs []int             `json:"disabledBuiltinIds"`
	// NextID 下一個要發的編號。必須一起存檔且只增不減 ——
	// 如果改用「現有最大值 +1」，刪掉最後一題之後新題目會拿到同一個編號，
	// 而示範圖是以題號當檔名，新題目就可能接到上一題留下來的圖。
	NextID int `json:"nextId"`
}

// QuestionStore 後台自訂的題目與停用清單。
//
// 跟示範圖一樣寫到磁碟而不是記憶體：這是「內容設定」，重開服務之後應該還在。
// 用一個 JSON 檔就夠了 —— 題目數量是幾百筆等級，引進資料庫只會讓部署變複雜。
type QuestionStore struct {
	mu   sync.RWMutex
	path string

	custom   []models.Question
	disabled map[int]bool
	nextID   int
}

// NewQuestionStore 開啟（或建立）題目設定檔
func NewQuestionStore(dir string) (*QuestionStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("建立題目資料夾失敗: %w", err)
	}

	s := &QuestionStore{
		path:     filepath.Join(dir, "questions.json"),
		custom:   make([]models.Question, 0),
		disabled: make(map[int]bool),
		nextID:   CustomQuestionIDBase,
	}

	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *QuestionStore) load() error {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil // 還沒有人新增過題目，空的就好
	}
	if err != nil {
		return fmt.Errorf("讀取題目設定失敗: %w", err)
	}

	var data questionFile
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("題目設定格式錯誤: %w", err)
	}

	s.custom = data.CustomQuestions
	if s.custom == nil {
		s.custom = make([]models.Question, 0)
	}
	for _, id := range data.DisabledBuiltinIDs {
		s.disabled[id] = true
	}

	s.nextID = data.NextID
	// 舊版設定檔沒有 nextId，用現有最大編號補回來，至少不會撞到還在的題目
	for _, q := range s.custom {
		if q.ID >= s.nextID {
			s.nextID = q.ID + 1
		}
	}
	if s.nextID < CustomQuestionIDBase {
		s.nextID = CustomQuestionIDBase
	}
	return nil
}

// saveLocked 寫檔。先寫暫存檔再 rename，避免寫到一半當掉留下壞檔案。
func (s *QuestionStore) saveLocked() error {
	disabled := make([]int, 0, len(s.disabled))
	for id, off := range s.disabled {
		if off {
			disabled = append(disabled, id)
		}
	}
	sort.Ints(disabled)

	raw, err := json.MarshalIndent(questionFile{
		CustomQuestions:    s.custom,
		DisabledBuiltinIDs: disabled,
		NextID:             s.nextID,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化題目設定失敗: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("寫入題目設定失敗: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("覆寫題目設定失敗: %w", err)
	}
	return nil
}

// Count 目前有幾題自訂題目
func (s *QuestionStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.custom)
}

// Path 設定檔的實際位置
func (s *QuestionStore) Path() string { return s.path }

// Custom 取得某模式的自訂題目
func (s *QuestionStore) Custom(mode models.GameMode) []models.Question {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]models.Question, 0, len(s.custom))
	for _, q := range s.custom {
		if q.Mode == mode {
			out = append(out, q)
		}
	}
	return out
}

// IsDisabled 這題是不是被後台停用了
func (s *QuestionStore) IsDisabled(id int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.disabled[id]
}

// DisabledIDs 目前被停用的題目編號
func (s *QuestionStore) DisabledIDs() []int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]int, 0, len(s.disabled))
	for id, off := range s.disabled {
		if off {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	return ids
}

func sanitizeQuestionText(text string) string {
	cleaned := strings.TrimSpace(strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(text))
	if runes := []rune(cleaned); len(runes) > 80 {
		cleaned = string(runes[:80])
	}
	return cleaned
}

func normalizeCategory(category string, mode models.GameMode) string {
	known := map[string]bool{
		models.CategoryExpression: true,
		models.CategoryImitate:    true,
		models.CategoryPose:       true,
		models.CategoryColor:      true,
		models.CategoryObject:     true,
		models.CategoryAdvanced:   true,
		models.CategoryGroup:      true,
		models.CategoryCustom:     true,
	}
	if known[category] {
		return category
	}
	if mode == models.ModeGroup {
		return models.CategoryGroup
	}
	return models.CategoryCustom
}

// Add 新增一題，回傳含編號的完整題目
func (s *QuestionStore) Add(text string, mode models.GameMode, category string, difficulty int) (models.Question, error) {
	cleaned := sanitizeQuestionText(text)
	if cleaned == "" {
		return models.Question{}, ErrQuestionTextRequired
	}

	if mode != models.ModeGroup {
		mode = models.ModeSolo
	}
	if difficulty != 2 {
		difficulty = 1
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	question := models.Question{
		ID:         s.nextID,
		Text:       cleaned,
		Category:   normalizeCategory(category, mode),
		Difficulty: difficulty,
		Mode:       mode,
		IsCustom:   true,
	}

	s.custom = append(s.custom, question)
	s.nextID++

	if err := s.saveLocked(); err != nil {
		s.custom = s.custom[:len(s.custom)-1]
		s.nextID--
		return models.Question{}, err
	}
	return question, nil
}

// Update 修改自訂題目的內容
func (s *QuestionStore) Update(id int, text, category string, difficulty int) (models.Question, error) {
	cleaned := sanitizeQuestionText(text)
	if cleaned == "" {
		return models.Question{}, ErrQuestionTextRequired
	}
	if id < CustomQuestionIDBase {
		return models.Question{}, ErrBuiltinNotEditable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, q := range s.custom {
		if q.ID != id {
			continue
		}

		before := q
		s.custom[i].Text = cleaned
		s.custom[i].Category = normalizeCategory(category, q.Mode)
		if difficulty == 2 {
			s.custom[i].Difficulty = 2
		} else {
			s.custom[i].Difficulty = 1
		}

		if err := s.saveLocked(); err != nil {
			s.custom[i] = before
			return models.Question{}, err
		}
		return s.custom[i], nil
	}

	return models.Question{}, ErrQuestionNotFound
}

// Delete 刪掉一題自訂題目
func (s *QuestionStore) Delete(id int) error {
	if id < CustomQuestionIDBase {
		return ErrBuiltinNotEditable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, q := range s.custom {
		if q.ID != id {
			continue
		}

		removed := s.custom
		s.custom = append(append([]models.Question{}, s.custom[:i]...), s.custom[i+1:]...)
		if err := s.saveLocked(); err != nil {
			s.custom = removed
			return err
		}
		return nil
	}

	return ErrQuestionNotFound
}

// SetDisabled 停用或啟用一題。內建題不能刪，但可以停用讓它不再被抽到。
func (s *QuestionStore) SetDisabled(id int, disabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := s.disabled[id]
	if disabled {
		s.disabled[id] = true
	} else {
		delete(s.disabled, id)
	}

	if err := s.saveLocked(); err != nil {
		if before {
			s.disabled[id] = true
		} else {
			delete(s.disabled, id)
		}
		return err
	}
	return nil
}
