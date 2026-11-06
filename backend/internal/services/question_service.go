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
// categories 不為空時只從這些分類抽，而且各分類輪流出題、數量平均；
// 只照整個題庫隨機抽的話，題數多的分類（例如表情）會把題數少的分類淹掉。
// 題庫不夠時會循環補題，確保一定湊得到 count 題。
func (s *QuestionService) BuildRandom(mode models.GameMode, count int, difficulty string, categories []string) []models.Question {
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

	if groups := groupByCategory(pool, categories); len(groups) > 0 {
		return drawBalanced(groups, count)
	}

	return drawCycling(pool, count)
}

// groupByCategory 依勾選的分類分組，順序照勾選順序。勾到的分類在題庫裡沒題目就略過，
// 全部都沒題目時回 nil，讓呼叫端退回整個題庫抽。
func groupByCategory(pool []models.Question, categories []string) [][]models.Question {
	if len(categories) == 0 {
		return nil
	}

	byCategory := make(map[string][]models.Question)
	for _, q := range pool {
		byCategory[q.Category] = append(byCategory[q.Category], q)
	}

	seen := make(map[string]bool)
	var groups [][]models.Question
	for _, c := range categories {
		if seen[c] || len(byCategory[c]) == 0 {
			continue
		}
		seen[c] = true
		groups = append(groups, byCategory[c])
	}
	return groups
}

// drawBalanced 每一輪從每個分類各抽一題（分類順序每輪重洗），
// 所以 10 題、3 個分類會是 4/3/3，不會有分類被抽到 0 題。
func drawBalanced(groups [][]models.Question, count int) []models.Question {
	decks := make([][]models.Question, len(groups))
	next := make([]int, len(groups))
	for i, g := range groups {
		decks[i] = append([]models.Question(nil), g...)
		rand.Shuffle(len(decks[i]), func(a, b int) { decks[i][a], decks[i][b] = decks[i][b], decks[i][a] })
	}

	order := make([]int, len(groups))
	for i := range order {
		order[i] = i
	}

	out := make([]models.Question, 0, count)
	for len(out) < count {
		rand.Shuffle(len(order), func(a, b int) { order[a], order[b] = order[b], order[a] })
		for _, gi := range order {
			if len(out) == count {
				break
			}
			deck := decks[gi]
			if next[gi] == len(deck) {
				// 這個分類抽完了就重洗再循環，跟整個題庫不夠時的處理一致
				rand.Shuffle(len(deck), func(a, b int) { deck[a], deck[b] = deck[b], deck[a] })
				next[gi] = 0
			}
			out = append(out, deck[next[gi]])
			next[gi]++
		}
	}
	return out
}

// drawCycling 整個題庫洗牌依序取，不夠就再洗一次接著補
func drawCycling(pool []models.Question, count int) []models.Question {
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
