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
	// DefaultMaxWinners 建房沒指定時，每題最多可以選幾張得獎
	DefaultMaxWinners = 5
	// DefaultResultSeconds 公布得獎畫面預設停留秒數，時間到自動下一題
	DefaultResultSeconds = 2
	// PointsLowerRanks 第 6 名以後的得獎分：名額開到全部時，被點到的人還是要比沒被點到的多一點
	PointsLowerRanks = 10
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
	// ErrNoQuestions 房間沒有題目
	ErrNoQuestions = errors.New("這個房間還沒有任何題目")
	// ErrNotJudging 目前不是評選階段
	ErrNotJudging = errors.New("現在不是評選階段")
	// ErrInvalidWinners 得獎張數不合法
	ErrInvalidWinners = errors.New("得獎張數不合法，超過這間房設定的名額")
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

	// 有開試玩就先跑一輪不計分的暖身，讓大家確認相機沒問題再進正式第 1 題
	if room.PracticeEnabled {
		room.InPractice = true
		room.RoundPhotos = make([]*models.PhotoSubmission, 0, 8)
		room.RoundStartAt = now
		room.RoundSeq++
		// 試玩不倒數，由房主決定什麼時候開始正式遊戲
		room.TimeLeft = 0
		room.Status = models.RoomStatusShooting
		room.LastActivity = now
		return nil
	}

	s.beginRoundLocked(room)
	return nil
}

// EndPractice 結束試玩，進入正式第 1 題
func (s *GameService) EndPractice(room *models.Room) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !room.InPractice {
		return errors.New("現在不是試玩回合")
	}

	room.InPractice = false
	s.beginRoundLocked(room)
	return nil
}

// beginRoundLocked 推進到下一題並開始倒數；呼叫前必須已持有鎖
func (s *GameService) beginRoundLocked(room *models.Room) {
	room.InPractice = false
	room.CurrentQuestion++
	room.RoundPhotos = make([]*models.PhotoSubmission, 0, 8)
	room.RoundStartAt = time.Now()
	room.RoundSeq++
	room.Advancing = false
	room.TimeLeft = room.QuestionTimeLimit
	room.Status = models.RoomStatusShooting
	room.LastActivity = time.Now()
}

// ClaimAdvance 從「公布得獎」畫面換題前先搶這一題的換題權。
// 自動計時與房主按「下一題」可能同時發生，只有先搶到的那一方會真的換題，
// 否則會連跳兩題。seq 傳 -1 代表不比對回合（房主手動）。
func (s *GameService) ClaimAdvance(room *models.Room, seq int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room.Status != models.RoomStatusRoundResult || room.Advancing {
		return false
	}
	if seq >= 0 && room.RoundSeq != seq {
		return false
	}
	room.Advancing = true
	return true
}

// Seq 目前的回合序號，給延遲動作比對「是不是還在同一題」
func (s *GameService) Seq(room *models.Room) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return room.RoundSeq
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

	// 不分模式，每人每題就是一張。再傳就是重拍覆蓋，沿用原本的抵達順序，
	// 也不重複給參與分 —— 否則一個人連傳好幾張就能洗版兼刷分。
	for i, existing := range room.RoundPhotos {
		if existing.PlayerID != player.ID {
			continue
		}
		replacedPhotoID = existing.PhotoID
		existing.PhotoID = photoID
		existing.URL = url
		existing.SubmittedAt = time.Now()
		room.LastActivity = existing.SubmittedAt
		return room.RoundPhotos[i], replacedPhotoID, nil
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

	// 試玩回合純粹暖身，不計分也不累計投稿數
	if room.InPractice {
		return sub, "", nil
	}

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

	// 試玩不自動收桌：讓每個人都有機會試拍，由房主決定何時開始
	if room.InPractice || room.Mode != models.ModeSolo {
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

// PointsForRank 第 rank 名（1 起算）的得獎分
func PointsForRank(rank int) int {
	if rank >= 1 && rank <= len(winPoints) {
		return winPoints[rank-1]
	}
	return PointsLowerRanks
}

// ApplyWinners 套用房主的評選結果並計分
func (s *GameService) ApplyWinners(room *models.Room, photoIDs []string) (models.RoundResult, []models.ScoreInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room.Status != models.RoomStatusJudging {
		return models.RoundResult{}, nil, ErrNotJudging
	}
	if len(photoIDs) == 0 || (room.MaxWinners > 0 && len(photoIDs) > room.MaxWinners) {
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
		points := PointsForRank(rank)

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
	room.Advancing = false
	room.CurrentQuestion = -1
	room.InPractice = false
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
