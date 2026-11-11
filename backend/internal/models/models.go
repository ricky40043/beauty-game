package models

import (
	"sort"
	"time"
)

// GameMode 遊戲模式
type GameMode string

const (
	ModeSolo  GameMode = "solo"  // 單人：每人各自拍一張
	ModeGroup GameMode = "group" // 團體：全場合體照，任何人都能代表上傳
)

// RoomStatus 房間狀態
type RoomStatus string

const (
	RoomStatusWaiting     RoomStatus = "waiting"      // 大廳等待
	RoomStatusShooting    RoomStatus = "shooting"     // 拍照中（倒數進行）
	RoomStatusJudging     RoomStatus = "judging"      // 房主評選中
	RoomStatusVoting      RoomStatus = "voting"       // 投票模式：玩家投票倒數中
	RoomStatusRoundResult RoomStatus = "round_result" // 公布本題得獎
	RoomStatusFinished    RoomStatus = "finished"     // 全部題目結束
	RoomStatusAbandoned   RoomStatus = "abandoned"    // 全員離線，等待清理
)

// QuestionCategory 題目分類
const (
	CategoryExpression = "expression" // 表情
	CategoryImitate    = "imitate"    // 模仿
	CategoryPose       = "pose"       // 自拍姿勢
	CategoryColor      = "color"      // 找顏色
	CategoryObject     = "object"     // 找物品
	CategoryAdvanced   = "advanced"   // 進階複合
	CategoryStudent    = "student"    // 學生篇
	CategoryWork       = "work"       // 職場篇
	CategoryChurch     = "church"     // 教會
	CategoryChristmas  = "christmas"  // 聖誕市集
	CategoryGroup      = "group"      // 團體合體照
	CategoryCustom     = "custom"     // 房主自訂
)

// Question 題目
type Question struct {
	ID         int      `json:"id"`
	Text       string   `json:"text"`
	Category   string   `json:"category"`
	Difficulty int      `json:"difficulty"` // 1 基礎、2 進階
	Mode       GameMode `json:"mode"`
	IsCustom   bool     `json:"isCustom"`
}

// Player 玩家（房主不算玩家，房主只負責主畫面與評選）
type Player struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Avatar       string    `json:"avatar"` // emoji
	RoomID       string    `json:"roomId"`
	Score        int       `json:"score"`
	Wins         int       `json:"wins"`    // 累計得獎次數
	Uploads      int       `json:"uploads"` // 累計成功投稿數
	IsConnected  bool      `json:"isConnected"`
	JoinedAt     time.Time `json:"joinedAt"`
	LastActivity time.Time `json:"lastActivity"`
}

// PhotoSubmission 一張本回合的投稿
type PhotoSubmission struct {
	PhotoID       string    `json:"photoId"`
	URL           string    `json:"url"`
	RoomID        string    `json:"roomId"`
	QuestionIndex int       `json:"questionIndex"`
	PlayerID      string    `json:"playerId"`
	PlayerName    string    `json:"playerName"`
	PlayerAvatar  string    `json:"playerAvatar"`
	Order         int       `json:"order"`      // 抵達順序，1 起算
	ElapsedSec    float64   `json:"elapsedSec"` // 從出題到上傳花的秒數
	SubmittedAt   time.Time `json:"submittedAt"`
}

// RoundWinner 本題得獎者
type RoundWinner struct {
	PhotoID      string `json:"photoId"`
	URL          string `json:"url"`
	PlayerID     string `json:"playerId"`
	PlayerName   string `json:"playerName"`
	PlayerAvatar string `json:"playerAvatar"`
	Rank         int    `json:"rank"` // 1~5，房主點選的順序
	Points       int    `json:"points"`
	Votes        int    `json:"votes,omitempty"` // 投票模式拿到的票數
}

// RoundResult 一題的完整結果，供結算頁照片牆使用
type RoundResult struct {
	QuestionIndex int           `json:"questionIndex"`
	QuestionText  string        `json:"questionText"`
	TotalPhotos   int           `json:"totalPhotos"`
	Winners       []RoundWinner `json:"winners"`
	GroupBonus    int           `json:"groupBonus"` // 團體模式全場合作分
}

// Room 房間
type Room struct {
	ID            string     `json:"id"`
	HostID        string     `json:"hostId"`
	HostName      string     `json:"hostName"`
	HostToken     string     `json:"hostToken"`
	HostConnected bool       `json:"hostConnected"`
	Mode          GameMode   `json:"mode"`
	Status        RoomStatus `json:"status"`

	// RequireNickname 為 false（預設）時，玩家掃碼即可入場，由伺服器自動配暱稱
	RequireNickname bool `json:"requireNickname"`

	// PracticeEnabled 開局先跑一輪試玩：不計分、不限張數顯示，讓大家先確認相機沒問題
	PracticeEnabled bool `json:"practiceEnabled"`
	// InPractice 目前正處於試玩回合
	InPractice bool `json:"inPractice"`
	// PracticeText 試玩回合的題目，空字串代表用該模式的預設題（見 DefaultPracticeText）
	PracticeText string `json:"practiceText"`

	// IntroEnabled 開始遊戲前主畫面先播一段玩法說明。播放本身由主畫面負責（房間仍在 waiting），
	// 伺服器只記住這個設定並把翻頁轉播給玩家手機
	IntroEnabled bool `json:"introEnabled"`

	Players   map[string]*Player `json:"players"`
	Questions []Question         `json:"questions"`

	// MaxWinners 每題最多可以選幾張得獎，0 代表不限（投稿幾張就能排幾名）
	MaxWinners int `json:"maxWinners"`

	// ResultSeconds 公布得獎後停幾秒自動進下一題，0 代表等房主按「下一題」
	ResultSeconds int `json:"resultSeconds"`
	// VoteMode 投票模式：最快的前 VoteCandidates 張是候選，玩家用手機投票（不能投自己），
	// 湊滿就停止收照片、開始 VoteSeconds 秒的投票倒數，依票數排名給分
	VoteMode       bool `json:"voteMode"`
	VoteCandidates int  `json:"voteCandidates"`
	VoteSeconds    int  `json:"voteSeconds"`
	// Votes 本題的投票：投票者 playerId → 被投的照片主人 playerId。
	// 用照片主人而不是 photoId 記，重拍換照片時票不會不見
	Votes map[string]string `json:"-"`

	// Advancing 這題的結果畫面已經有人（自動計時或房主）負責換題了，另一方不要再換一次
	Advancing bool `json:"-"`

	// QuestionMode "random" | "custom"。隨機出題的房間「再來一局」會重抽，自選的維持原題
	QuestionMode string `json:"questionMode"`
	// RandomDifficulty 隨機抽題的難度設定，再來一局重抽時沿用
	RandomDifficulty string `json:"-"`
	// PlayedQuestionIDs 這間房玩過的題目，重抽時優先避開，才不會下一局又是同一批
	PlayedQuestionIDs map[int]bool `json:"-"`

	// RandomCategories 建房時勾的抽題分類，大廳「重新隨機」要沿用，不然會洗回全部分類
	RandomCategories []string `json:"randomCategories"`

	CurrentQuestion   int `json:"currentQuestion"` // 0 起算；-1 表示尚未開始
	QuestionTimeLimit int `json:"questionTimeLimit"`
	TimeLeft          int `json:"timeLeft"`

	RoundPhotos  []*PhotoSubmission `json:"roundPhotos"` // 本回合投稿，依抵達順序
	RoundStartAt time.Time          `json:"roundStartAt"`
	RoundSeq     int                `json:"roundSeq"` // 每開一回合 +1，用來讓過期的計時器自動失效

	History []RoundResult `json:"history"`

	CreatedAt    time.Time  `json:"createdAt"`
	LastActivity time.Time  `json:"lastActivity"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	FinishedAt   *time.Time `json:"finishedAt,omitempty"`
}

// ScoreInfo 排行榜項目
type ScoreInfo struct {
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	Avatar     string `json:"avatar"`
	Score      int    `json:"score"`
	Rank       int    `json:"rank"`
	Wins       int    `json:"wins"`
	Uploads    int    `json:"uploads"`
	Gained     int    `json:"gained"` // 本題得分
}

// CreateRoomRequest 建房參數（WebSocket CREATE_ROOM 的 payload）
type CreateRoomRequest struct {
	HostName          string   `json:"hostName"`
	Mode              GameMode `json:"mode"`
	TotalQuestions    int      `json:"totalQuestions"`
	QuestionTimeLimit int      `json:"questionTimeLimit"`
	Difficulty        string   `json:"difficulty"`      // "basic" | "mixed"
	QuestionMode      string   `json:"questionMode"`    // "random" | "custom"
	QuestionIDs       []int    `json:"questionIds"`     // 自選模式：題庫題目，順序即遊玩順序
	CustomQuestions   []string `json:"customQuestions"` // 自選模式：房主自己打的題目，接在後面
	Categories        []string `json:"categories"`      // 隨機模式：只從這些分類平均抽；空的代表全部
	MaxWinners        *int     `json:"maxWinners"`      // 每題得獎名額，0 = 不限；沒帶就用預設 5
	ResultSeconds     *int     `json:"resultSeconds"`   // 得獎畫面停幾秒自動下一題，0 = 手動；沒帶就用預設 2
	VoteMode          *bool    `json:"voteMode"`        // 投票模式：玩家投票決定名次
	VoteCandidates    *int     `json:"voteCandidates"`  // 投票模式的候選張數（最快的前 N 張）
	VoteSeconds       *int     `json:"voteSeconds"`     // 投票倒數秒數
	RequireNickname   bool     `json:"requireNickname"`
	PracticeRound     bool     `json:"practiceRound"`
	PracticeText      string   `json:"practiceText"` // 試玩題目，空的就用該模式的預設題
	GameIntro         bool     `json:"gameIntro"`    // 開始前先播玩法說明
}

// PracticeQuestion 試玩回合用的題目。不屬於題庫，也不計入總題數。
// 實際顯示的文字由房間決定，見 Room.PracticeQuestionData。
var PracticeQuestion = Question{
	ID:         9999,
	Text:       "試玩：隨便拍一張，確認相機沒問題",
	Category:   CategoryCustom,
	Difficulty: 1,
}

// DefaultPracticeText 主持人沒指定試玩題時用的題目：要簡單到不用想，重點是確認相機沒問題
func DefaultPracticeText(mode GameMode) string {
	if mode == ModeGroup {
		return "全員一起比 YA"
	}
	return "比出小愛心"
}

// PracticeQuestionData 這間房的試玩題
func (r *Room) PracticeQuestionData() Question {
	q := PracticeQuestion
	q.Mode = r.Mode
	q.Text = r.PracticeText
	if q.Text == "" {
		q.Text = DefaultPracticeText(r.Mode)
	}
	return q
}

// AddPlayer 新增玩家
func (r *Room) AddPlayer(p *Player) {
	if r.Players == nil {
		r.Players = make(map[string]*Player)
	}
	r.Players[p.ID] = p
}

// GetPlayer 取得玩家
func (r *Room) GetPlayer(id string) (*Player, bool) {
	p, ok := r.Players[id]
	return p, ok
}

// RemovePlayer 移除玩家
func (r *Room) RemovePlayer(id string) {
	delete(r.Players, id)
}

// PlayerList 依加入時間排序的玩家陣列
func (r *Room) PlayerList() []*Player {
	list := make([]*Player, 0, len(r.Players))
	for _, p := range r.Players {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].JoinedAt.Before(list[j].JoinedAt)
	})
	return list
}

// ConnectedPlayerCount 線上玩家數
func (r *Room) ConnectedPlayerCount() int {
	n := 0
	for _, p := range r.Players {
		if p.IsConnected {
			n++
		}
	}
	return n
}

// CurrentQuestionData 取得目前題目
func (r *Room) CurrentQuestionData() (Question, bool) {
	if r.CurrentQuestion < 0 || r.CurrentQuestion >= len(r.Questions) {
		return Question{}, false
	}
	return r.Questions[r.CurrentQuestion], true
}

// HasSubmitted 判斷玩家本回合是否已投稿
func (r *Room) HasSubmitted(playerID string) bool {
	for _, p := range r.RoundPhotos {
		if p.PlayerID == playerID {
			return true
		}
	}
	return false
}

// SubmissionCountOf 玩家本回合的投稿張數
func (r *Room) SubmissionCountOf(playerID string) int {
	n := 0
	for _, p := range r.RoundPhotos {
		if p.PlayerID == playerID {
			n++
		}
	}
	return n
}

// Leaderboard 依分數排序的排行榜
func (r *Room) Leaderboard() []ScoreInfo {
	scores := make([]ScoreInfo, 0, len(r.Players))
	for _, p := range r.Players {
		scores = append(scores, ScoreInfo{
			PlayerID:   p.ID,
			PlayerName: p.Name,
			Avatar:     p.Avatar,
			Score:      p.Score,
			Wins:       p.Wins,
			Uploads:    p.Uploads,
		})
	}

	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Score != scores[j].Score {
			return scores[i].Score > scores[j].Score
		}
		if scores[i].Wins != scores[j].Wins {
			return scores[i].Wins > scores[j].Wins
		}
		return scores[i].PlayerName < scores[j].PlayerName
	})

	// 同分同名次（1、1、3），同分的第一名要一起當冠軍
	for i := range scores {
		if i > 0 && scores[i].Score == scores[i-1].Score {
			scores[i].Rank = scores[i-1].Rank
		} else {
			scores[i].Rank = i + 1
		}
	}
	return scores
}
