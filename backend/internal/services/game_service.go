package services

import (
	"errors"
	"sync"
	"time"

	"beauty-game/internal/models"
)

// 計分規則
const (
	// PointsForUpload 完成上傳的參與分
	PointsForUpload = 10
	// PointsGroupBonus 團體模式該回合有得獎時，全場每人的合作分
	PointsGroupBonus = 50
	// MaxWinnersPerRound 房主每題最多可以選幾張得獎
	MaxWinnersPerRound = 5
	// MaxGroupUploadsPerPlayer 團體模式每人每回合的投稿上限
	MaxGroupUploadsPerPlayer = 3
)

var (
	// speedBonus 抵達順序前三名的速度分
	speedBonus = []int{15, 10, 5}
	// winPoints 得獎分數，依房主點選順序
	winPoints = []int{100, 80, 60, 40, 20}
)

var (
	// ErrNotShooting 目前不是拍照階段
	ErrNotShooting = errors.New("現在不是拍照時間")
	// ErrAlreadySubmitted 單人模式已投稿過
	ErrAlreadySubmitted = errors.New("這題你已經上傳過了")
	// ErrUploadLimit 團體模式投稿數已達上限
	ErrUploadLimit = errors.New("這題你的上傳次數已達上限")
	// ErrNoQuestions 房間沒有題目
	ErrNoQuestions = errors.New("這個房間還沒有任何題目")
	// ErrNotJudging 目前不是評選階段
	ErrNotJudging = errors.New("現在不是評選階段")
	// ErrInvalidWinners 得獎張數不合法
	ErrInvalidWinners = errors.New("請選 1~5 張照片")
)

// GameService 掌管回合狀態機與計分。所有會改動房間的動作都經過這裡的鎖。
type GameService struct {
	mu sync.Mutex
}

// NewGameService 建立遊戲服務
func NewGameService() *GameService {
	return &GameService{}
}

// Lock/Unlock 讓 WebSocket 層在讀取房間快照時也能取得一致狀態
func (s *GameService) Lock()   { s.mu.Lock() }
func (s *GameService) Unlock() { s.mu.Unlock() }

// StartGame 開始遊戲並進入第一題
func (s *GameService) StartGame(room *models.Room) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(room.Questions) == 0 {
		return ErrNoQuestions
	}
	if room.Status != models.RoomStatusWaiting {
		return errors.New("遊戲已經開始了")
	}

	now := time.Now()
	room.StartedAt = &now
	room.CurrentQuestion = -1
	room.History = room.History[:0]

	for _, p := range room.Players {
		p.Score = 0
		p.Wins = 0
		p.Uploads = 0
	}

	s.beginRoundLocked(room)
	return nil
}

// beginRoundLocked 推進到下一題並開始倒數；呼叫前必須已持有鎖
func (s *GameService) beginRoundLocked(room *models.Room) {
	room.CurrentQuestion++
	room.RoundPhotos = make([]*models.PhotoSubmission, 0, 8)
	room.RoundStartAt = time.Now()
	room.RoundSeq++
	room.TimeLeft = room.QuestionTimeLimit
	room.Status = models.RoomStatusShooting
	room.LastActivity = time.Now()
}

// NextRound 進入下一題。回傳 false 表示題目已經出完。
func (s *GameService) NextRound(room *models.Room) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room.CurrentQuestion+1 >= len(room.Questions) {
		now := time.Now()
		room.FinishedAt = &now
		room.Status = models.RoomStatusFinished
		room.TimeLeft = 0
		room.LastActivity = now
		return false
	}

	s.beginRoundLocked(room)
	return true
}

// Tick 每秒扣一次倒數。回傳剩餘秒數與「是否該收桌」。
// seq 用來讓上一題殘留的計時器自動失效。
func (s *GameService) Tick(room *models.Room, seq int) (timeLeft int, expired bool, stale bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room.RoundSeq != seq || room.Status != models.RoomStatusShooting {
		return room.TimeLeft, false, true
	}

	room.TimeLeft--
	if room.TimeLeft <= 0 {
		room.TimeLeft = 0
		return 0, true, false
	}
	return room.TimeLeft, false, false
}

// AddSubmission 記錄一張投稿並給參與分與速度分。
// 單人模式重複上傳會覆蓋舊照片，回傳的 replacedPhotoID 需要由呼叫端從照片庫刪掉。
func (s *GameService) AddSubmission(room *models.Room, player *models.Player, photoID, url string) (sub *models.PhotoSubmission, replacedPhotoID string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room.Status != models.RoomStatusShooting {
		return nil, "", ErrNotShooting
	}

	if room.Mode == models.ModeSolo {
		for i, existing := range room.RoundPhotos {
			if existing.PlayerID != player.ID {
				continue
			}
			// 重拍：沿用原本的抵達順序，只換照片，不重複給參與分
			replacedPhotoID = existing.PhotoID
			existing.PhotoID = photoID
			existing.URL = url
			existing.SubmittedAt = time.Now()
			room.LastActivity = existing.SubmittedAt
			return room.RoundPhotos[i], replacedPhotoID, nil
		}
	} else if room.SubmissionCountOf(player.ID) >= MaxGroupUploadsPerPlayer {
		return nil, "", ErrUploadLimit
	}

	now := time.Now()
	order := len(room.RoundPhotos) + 1
	question, _ := room.CurrentQuestionData()

	sub = &models.PhotoSubmission{
		PhotoID:       photoID,
		URL:           url,
		RoomID:        room.ID,
		QuestionIndex: room.CurrentQuestion,
		PlayerID:      player.ID,
		PlayerName:    player.Name,
		PlayerAvatar:  player.Avatar,
		Order:         order,
		ElapsedSec:    now.Sub(room.RoundStartAt).Seconds(),
		SubmittedAt:   now,
	}
	_ = question

	room.RoundPhotos = append(room.RoundPhotos, sub)
	room.LastActivity = now

	player.Uploads++
	player.Score += PointsForUpload
	if order <= len(speedBonus) {
		player.Score += speedBonus[order-1]
	}

	return sub, "", nil
}

// AllPlayersSubmitted 單人模式下，線上玩家是否都投稿完了
func (s *GameService) AllPlayersSubmitted(room *models.Room) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room.Mode != models.ModeSolo {
		return false
	}

	connected := 0
	for _, p := range room.Players {
		if !p.IsConnected {
			continue
		}
		connected++
		if !room.HasSubmitted(p.ID) {
			return false
		}
	}

	return connected > 0
}

// CloseRound 收桌，進入評選階段。回傳本回合的投稿。
func (s *GameService) CloseRound(room *models.Room) []*models.PhotoSubmission {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room.Status != models.RoomStatusShooting {
		return room.RoundPhotos
	}

	room.Status = models.RoomStatusJudging
	room.TimeLeft = 0
	room.LastActivity = time.Now()

	return room.RoundPhotos
}

// ApplyWinners 套用房主的評選結果並計分
func (s *GameService) ApplyWinners(room *models.Room, photoIDs []string) (models.RoundResult, []models.ScoreInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room.Status != models.RoomStatusJudging {
		return models.RoundResult{}, nil, ErrNotJudging
	}
	if len(photoIDs) == 0 || len(photoIDs) > MaxWinnersPerRound {
		return models.RoundResult{}, nil, ErrInvalidWinners
	}

	byPhotoID := make(map[string]*models.PhotoSubmission, len(room.RoundPhotos))
	for _, p := range room.RoundPhotos {
		byPhotoID[p.PhotoID] = p
	}

	question, _ := room.CurrentQuestionData()
	result := models.RoundResult{
		QuestionIndex: room.CurrentQuestion,
		QuestionText:  question.Text,
		TotalPhotos:   len(room.RoundPhotos),
		Winners:       make([]models.RoundWinner, 0, len(photoIDs)),
	}

	gained := make(map[string]int)
	seen := make(map[string]bool)

	for _, photoID := range photoIDs {
		sub, ok := byPhotoID[photoID]
		if !ok || seen[photoID] {
			continue
		}
		seen[photoID] = true

		rank := len(result.Winners) + 1
		points := winPoints[rank-1]

		result.Winners = append(result.Winners, models.RoundWinner{
			PhotoID:      sub.PhotoID,
			URL:          sub.URL,
			PlayerID:     sub.PlayerID,
			PlayerName:   sub.PlayerName,
			PlayerAvatar: sub.PlayerAvatar,
			Rank:         rank,
			Points:       points,
		})

		if player, ok := room.Players[sub.PlayerID]; ok {
			player.Score += points
			player.Wins++
		}
		gained[sub.PlayerID] += points
	}

	if len(result.Winners) == 0 {
		return models.RoundResult{}, nil, ErrInvalidWinners
	}

	// 團體模式是合體照，功勞屬於全場：拍的人拿名次分，所有人再一起拿合作分
	if room.Mode == models.ModeGroup {
		result.GroupBonus = PointsGroupBonus
		for _, player := range room.Players {
			player.Score += PointsGroupBonus
			gained[player.ID] += PointsGroupBonus
		}
	}

	room.History = append(room.History, result)
	room.Status = models.RoomStatusRoundResult
	room.LastActivity = time.Now()

	scores := room.Leaderboard()
	for i := range scores {
		scores[i].Gained = gained[scores[i].PlayerID]
	}

	return result, scores, nil
}

// SkipRound 本題沒有任何投稿，或房主決定不選任何人
func (s *GameService) SkipRound(room *models.Room) models.RoundResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	question, _ := room.CurrentQuestionData()
	result := models.RoundResult{
		QuestionIndex: room.CurrentQuestion,
		QuestionText:  question.Text,
		TotalPhotos:   len(room.RoundPhotos),
		Winners:       []models.RoundWinner{},
	}

	room.History = append(room.History, result)
	room.Status = models.RoomStatusRoundResult
	room.LastActivity = time.Now()

	return result
}

// FinishGame 直接結束整場遊戲
func (s *GameService) FinishGame(room *models.Room) []models.ScoreInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	room.FinishedAt = &now
	room.Status = models.RoomStatusFinished
	room.TimeLeft = 0
	room.LastActivity = now

	return room.Leaderboard()
}

// ResetToLobby 再來一局：清空分數與紀錄回到大廳
func (s *GameService) ResetToLobby(room *models.Room) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room.Status = models.RoomStatusWaiting
	room.CurrentQuestion = -1
	room.RoundPhotos = nil
	room.RoundSeq++
	room.TimeLeft = 0
	room.History = room.History[:0]
	room.StartedAt = nil
	room.FinishedAt = nil
	room.LastActivity = time.Now()

	for _, p := range room.Players {
		p.Score = 0
		p.Wins = 0
		p.Uploads = 0
	}
}

// Leaderboard 取得目前排行榜快照
func (s *GameService) Leaderboard(room *models.Room) []models.ScoreInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return room.Leaderboard()
}
