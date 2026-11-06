package services

import (
	"errors"
	"sort"
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
	// DefaultVoteCandidates 投票模式預設候選張數
	DefaultVoteCandidates = 5
	// DefaultVoteSeconds 投票模式預設倒數秒數
	DefaultVoteSeconds = 10
	// MaxVoteSeconds 投票倒數上限
	MaxVoteSeconds = 25
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
	// ErrVoteClosed 現在不能投票
	ErrVoteClosed = errors.New("現在不能投票")
	// ErrVoteSelf 不能投給自己
	ErrVoteSelf = errors.New("不能投給自己喔")
	// ErrVoteTarget 投的照片不是候選
	ErrVoteTarget = errors.New("這張照片不能投")
	// ErrCandidatesFull 候選照片已滿
	ErrCandidatesFull = errors.New("候選照片已經滿了，等投票結果吧")
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
	room.Votes = make(map[string]string)
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
	return s.TickStatus(room, seq, models.RoomStatusShooting)
}

// TickStatus 同 Tick，但指定這個倒數屬於哪個階段（拍照或投票），換階段後舊倒數自動失效
func (s *GameService) TickStatus(room *models.Room, seq int, status models.RoomStatus) (timeLeft int, expired bool, stale bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room.RoundSeq != seq || room.Status != status {
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
	if s.votingLocked(room) && len(room.RoundPhotos) >= room.VoteCandidates && !room.HasSubmitted(player.ID) {
		return nil, "", ErrCandidatesFull
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

// votingLocked 這一題是否走投票模式（試玩不投票）
func (s *GameService) votingLocked(room *models.Room) bool {
	return room.VoteMode && !room.InPractice
}

// CandidatesFull 投票模式下候選照片是否已經湊滿，湊滿就該停止收照片開始投票
func (s *GameService) CandidatesFull(room *models.Room) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.votingLocked(room) && len(room.RoundPhotos) >= room.VoteCandidates
}

// CastVote 投票（targetID 空字串代表收回）。每人一票、可改投、不能投自己，
// 拍照中與投票倒數中都能投。回傳最新的各照片票數。
func (s *GameService) CastVote(room *models.Room, voterID, targetID string) (map[string]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.votingLocked(room) || (room.Status != models.RoomStatusShooting && room.Status != models.RoomStatusVoting) {
		return nil, ErrVoteClosed
	}
	if _, ok := room.Players[voterID]; !ok {
		return nil, ErrVoteClosed
	}
	if room.Votes == nil {
		room.Votes = make(map[string]string)
	}

	if targetID == "" {
		delete(room.Votes, voterID)
		return voteCountsLocked(room), nil
	}
	if targetID == voterID {
		return nil, ErrVoteSelf
	}
	if !room.HasSubmitted(targetID) {
		return nil, ErrVoteTarget
	}

	room.Votes[voterID] = targetID
	room.LastActivity = time.Now()
	return voteCountsLocked(room), nil
}

// VoteCounts 本題各照片主人的票數
func (s *GameService) VoteCounts(room *models.Room) map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return voteCountsLocked(room)
}

// MyVote 某位玩家本題投給誰（重連恢復畫面用）
func (s *GameService) MyVote(room *models.Room, voterID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return room.Votes[voterID]
}

func voteCountsLocked(room *models.Room) map[string]int {
	counts := make(map[string]int)
	for voter, target := range room.Votes {
		// 投票者或照片主人已經離開房間就不算
		if _, ok := room.Players[voter]; !ok {
			continue
		}
		if !room.HasSubmitted(target) {
			continue
		}
		counts[target]++
	}
	return counts
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
// 投票模式且有照片時進入投票倒數（voting = true），否則進房主評選。
func (s *GameService) CloseRound(room *models.Room) (photos []*models.PhotoSubmission, voting bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room.Status != models.RoomStatusShooting {
		return room.RoundPhotos, room.Status == models.RoomStatusVoting
	}

	room.LastActivity = time.Now()
	if s.votingLocked(room) && len(room.RoundPhotos) > 0 {
		room.Status = models.RoomStatusVoting
		room.TimeLeft = room.VoteSeconds
		return room.RoundPhotos, true
	}

	room.Status = models.RoomStatusJudging
	room.TimeLeft = 0
	return room.RoundPhotos, false
}

// rankedEntry 一張得獎照片與它的名次（同名次可以重複）
type rankedEntry struct {
	sub   *models.PhotoSubmission
	rank  int
	votes int
}

// applyRankedLocked 依排好的名次計分、寫入歷史並進入公布結果。呼叫前必須持有鎖。
func (s *GameService) applyRankedLocked(room *models.Room, entries []rankedEntry) (models.RoundResult, []models.ScoreInfo) {
	question, _ := room.CurrentQuestionData()
	result := models.RoundResult{
		QuestionIndex: room.CurrentQuestion,
		QuestionText:  question.Text,
		TotalPhotos:   len(room.RoundPhotos),
		Winners:       make([]models.RoundWinner, 0, len(entries)),
	}
	gained := make(map[string]int)

	for _, e := range entries {
		points := PointsForRank(e.rank)
		result.Winners = append(result.Winners, models.RoundWinner{
			PhotoID:      e.sub.PhotoID,
			URL:          e.sub.URL,
			PlayerID:     e.sub.PlayerID,
			PlayerName:   e.sub.PlayerName,
			PlayerAvatar: e.sub.PlayerAvatar,
			Rank:         e.rank,
			Points:       points,
			Votes:        e.votes,
		})
		if player, ok := room.Players[e.sub.PlayerID]; ok {
			player.Score += points
			player.Wins++
		}
		gained[e.sub.PlayerID] += points
	}

	// 團體模式是合體照，功勞屬於全場：拍的人拿名次分，所有人再一起拿合作分
	if room.Mode == models.ModeGroup && len(entries) > 0 {
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
	return result, scores
}

// ApplyVotes 投票倒數結束：依票數排名（同票同名次、同票再比誰先交），0 票不算得獎，
// 名次超過得獎名額的不給分。ok = false 代表沒有人得票，呼叫端應當成跳過這題。
func (s *GameService) ApplyVotes(room *models.Room) (result models.RoundResult, scores []models.ScoreInfo, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room.Status != models.RoomStatusVoting {
		return models.RoundResult{}, nil, false
	}

	counts := voteCountsLocked(room)
	candidates := make([]*models.PhotoSubmission, 0, len(room.RoundPhotos))
	for _, p := range room.RoundPhotos {
		if counts[p.PlayerID] > 0 {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return models.RoundResult{}, nil, false
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		ci, cj := counts[candidates[i].PlayerID], counts[candidates[j].PlayerID]
		if ci != cj {
			return ci > cj
		}
		return candidates[i].Order < candidates[j].Order
	})

	entries := make([]rankedEntry, 0, len(candidates))
	for i, p := range candidates {
		rank := i + 1
		if i > 0 && counts[p.PlayerID] == counts[candidates[i-1].PlayerID] {
			rank = entries[i-1].rank
		}
		if room.MaxWinners > 0 && rank > room.MaxWinners {
			break
		}
		entries = append(entries, rankedEntry{sub: p, rank: rank, votes: counts[p.PlayerID]})
	}

	result, scores = s.applyRankedLocked(room, entries)
	return result, scores, true
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

	entries := make([]rankedEntry, 0, len(photoIDs))
	seen := make(map[string]bool)
	for _, photoID := range photoIDs {
		sub, ok := byPhotoID[photoID]
		if !ok || seen[photoID] {
			continue
		}
		seen[photoID] = true
		entries = append(entries, rankedEntry{sub: sub, rank: len(entries) + 1})
	}

	if len(entries) == 0 {
		return models.RoundResult{}, nil, ErrInvalidWinners
	}

	result, scores := s.applyRankedLocked(room, entries)
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
	room.Votes = nil
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
