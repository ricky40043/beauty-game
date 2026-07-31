package services

import (
	"math/rand"
	"strings"

	"beauty-game/internal/models"
	"beauty-game/internal/names"
)

// QuestionService 負責出題：隨機抽題，或依房主指定的順序組題。
// 題庫 = 內建題（編譯在程式裡）+ 後台自訂題（存在磁碟），扣掉被停用的。
type QuestionService struct {
	store *QuestionStore
}

// NewQuestionService 建立出題服務。store 為 nil 時只有內建題。
func NewQuestionService(store *QuestionStore) *QuestionService {
	return &QuestionService{store: store}
}

// Store 取得底層的自訂題目儲存，給後台 handler 用
func (s *QuestionService) Store() *QuestionStore { return s.store }

// GetBank 取得某模式目前可用的完整題庫（內建 + 自訂 − 停用）
func (s *QuestionService) GetBank(mode models.GameMode) []models.Question {
	builtin := AllQuestions(mode)

	if s.store == nil {
		return builtin
	}

	out := make([]models.Question, 0, len(builtin))
	for _, q := range builtin {
		if !s.store.IsDisabled(q.ID) {
			out = append(out, q)
		}
	}
	for _, q := range s.store.Custom(mode) {
		if !s.store.IsDisabled(q.ID) {
			out = append(out, q)
		}
	}
	return out
}

// BuildRandom 隨機抽題。difficulty 為 "basic" 時只抽基礎題，其餘視為混合。
// 題庫不夠時會循環補題，確保一定湊得到 count 題。
func (s *QuestionService) BuildRandom(mode models.GameMode, count int, difficulty string) []models.Question {
	pool := s.GetBank(mode)

	if difficulty == "basic" {
		filtered := pool[:0:0]
		for _, q := range pool {
			if q.Difficulty == 1 {
				filtered = append(filtered, q)
			}
		}
		if len(filtered) > 0 {
			pool = filtered
		}
	}

	if count <= 0 {
		count = 10
	}
	if len(pool) == 0 {
		// 全部題目都被停用時的保險，至少讓遊戲開得起來
		return []models.Question{models.PracticeQuestion}
	}

	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	out := make([]models.Question, 0, count)
	for len(out) < count {
		remaining := count - len(out)
		if remaining >= len(pool) {
			out = append(out, pool...)
			// 還不夠就再洗一次牌繼續補，避免第二輪出現一樣的順序
			rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
			continue
		}
		out = append(out, pool[:remaining]...)
	}

	return out
}

// BuildCustom 依房主指定的題目與順序組題。
// questionIDs 是題庫題目（順序即遊玩順序），customTexts 是房主自己打的題目，接在後面。
func (s *QuestionService) BuildCustom(mode models.GameMode, questionIDs []int, customTexts []string) []models.Question {
	byID := make(map[int]models.Question)
	for _, q := range s.GetBank(mode) {
		byID[q.ID] = q
	}

	out := make([]models.Question, 0, len(questionIDs)+len(customTexts))
	seen := make(map[int]bool)

	for _, id := range questionIDs {
		q, ok := byID[id]
		if !ok || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, q)
	}

	nextCustomID := 9000
	for _, text := range customTexts {
		cleaned := strings.TrimSpace(text)
		if cleaned == "" {
			continue
		}
		if runes := []rune(cleaned); len(runes) > 60 {
			cleaned = string(runes[:60])
		}
		out = append(out, models.Question{
			ID:         nextCustomID,
			Text:       cleaned,
			Category:   models.CategoryCustom,
			Difficulty: 1,
			Mode:       mode,
			IsCustom:   true,
		})
		nextCustomID++
	}

	return out
}

// SanitizeName 暴露給其他層使用的暱稱清理（集中在同一處避免各自實作）
func SanitizeName(raw string) string {
	return names.Sanitize(raw)
}
