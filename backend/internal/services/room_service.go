package services

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"beauty-game/internal/config"
	"beauty-game/internal/models"
	"beauty-game/internal/names"

	"github.com/google/uuid"
)

var (
	// ErrRoomNotFound 房間不存在
	ErrRoomNotFound = errors.New("找不到這個房間")
	// ErrRoomFull 房間已滿
	ErrRoomFull = errors.New("房間人數已滿")
	// ErrNameRequired 房間要求自行取名，但沒帶名字
	ErrNameRequired = errors.New("這個房間需要輸入暱稱")
	// ErrNameTaken 暱稱重複
	ErrNameTaken = errors.New("這個暱稱已經有人用了")
)

// RoomService 純記憶體房間管理
type RoomService struct {
	mu    sync.RWMutex
	rooms map[string]*models.Room

	cfg       *config.Config
	questions *QuestionService
}

// NewRoomService 建立房間服務
func NewRoomService(cfg *config.Config, questions *QuestionService) *RoomService {
	return &RoomService{
		rooms:     make(map[string]*models.Room),
		cfg:       cfg,
		questions: questions,
	}
}

// CreateRoom 依建房參數開一間房。題目在這時候就決定好順序。
func (s *RoomService) CreateRoom(req models.CreateRoomRequest) (*models.Room, error) {
	mode := models.ModeSolo
	if req.Mode == models.ModeGroup {
		mode = models.ModeGroup
	}

	timeLimit := req.QuestionTimeLimit
	if timeLimit < 15 {
		timeLimit = s.cfg.Game.QuestionTimeLimit
	}
	if timeLimit > 300 {
		timeLimit = 300
	}

	var questions []models.Question
	if req.QuestionMode == "custom" {
		questions = s.questions.BuildCustom(mode, req.QuestionIDs, req.CustomQuestions)
	}
	// 自選模式沒選到任何題目時，退回隨機抽題，避免開出一間零題的房
	if len(questions) == 0 {
		total := req.TotalQuestions
		if total <= 0 {
			total = s.cfg.Game.DefaultTotalQuestions
		}
		if total > 50 {
			total = 50
		}
		questions = s.questions.BuildRandom(mode, total, req.Difficulty)
	}

	hostName := names.Sanitize(req.HostName)
	if hostName == "" {
		hostName = "主持人"
	}

	now := time.Now()
	room := &models.Room{
		ID:                s.generateRoomID(),
		HostName:          hostName,
		HostToken:         uuid.New().String(),
		HostConnected:     true,
		Mode:              mode,
		Status:            models.RoomStatusWaiting,
		RequireNickname:   req.RequireNickname,
		Players:           make(map[string]*models.Player),
		Questions:         questions,
		CurrentQuestion:   -1,
		QuestionTimeLimit: timeLimit,
		History:           make([]models.RoundResult, 0),
		CreatedAt:         now,
		LastActivity:      now,
	}

	s.mu.Lock()
	s.rooms[room.ID] = room
	s.mu.Unlock()

	return room, nil
}

// GetRoom 取得房間
func (s *RoomService) GetRoom(roomID string) (*models.Room, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	room, ok := s.rooms[strings.ToUpper(roomID)]
	if !ok {
		return nil, ErrRoomNotFound
	}
	return room, nil
}

// Exists 房間是否存在
func (s *RoomService) Exists(roomID string) bool {
	_, err := s.GetRoom(roomID)
	return err == nil
}

// AddPlayer 加入玩家。
// requestedName 為空且房間不要求取名時，自動配一組「形容詞 + 動物」暱稱。
func (s *RoomService) AddPlayer(roomID, playerID, requestedName string) (*models.Player, error) {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(room.Players) >= s.cfg.Game.MaxPlayersPerRoom {
		if _, already := room.Players[playerID]; !already {
			return nil, ErrRoomFull
		}
	}

	taken := make(map[string]bool, len(room.Players))
	for id, p := range room.Players {
		if id != playerID {
			taken[p.Name] = true
		}
	}

	name := names.Sanitize(requestedName)
	avatar := names.RandomAvatar()

	if name == "" {
		if room.RequireNickname {
			return nil, ErrNameRequired
		}
		name, avatar = names.Generate(taken)
	} else if taken[name] {
		return nil, ErrNameTaken
	}

	now := time.Now()

	// 同一個 playerID 重新加入時沿用既有分數，不重置
	if existing, ok := room.Players[playerID]; ok {
		existing.Name = name
		existing.IsConnected = true
		existing.LastActivity = now
		room.LastActivity = now
		return existing, nil
	}

	player := &models.Player{
		ID:           playerID,
		Name:         name,
		Avatar:       avatar,
		RoomID:       room.ID,
		IsConnected:  true,
		JoinedAt:     now,
		LastActivity: now,
	}
	room.AddPlayer(player)
	room.LastActivity = now

	return player, nil
}

// RenamePlayer 大廳改名（免取名模式下讓玩家可以自己換掉隨機暱稱）
func (s *RoomService) RenamePlayer(roomID, playerID, newName string) (*models.Player, error) {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	player, ok := room.Players[playerID]
	if !ok {
		return nil, errors.New("你不在這個房間裡")
	}

	name := names.Sanitize(newName)
	if name == "" {
		return nil, errors.New("暱稱不能是空的")
	}

	for id, p := range room.Players {
		if id != playerID && p.Name == name {
			return nil, ErrNameTaken
		}
	}

	player.Name = name
	player.LastActivity = time.Now()
	room.LastActivity = player.LastActivity

	return player, nil
}

// SetPlayerConnected 更新玩家連線狀態（斷線時保留玩家與分數，等他重連）
func (s *RoomService) SetPlayerConnected(roomID, playerID string, connected bool) {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if player, ok := room.Players[playerID]; ok {
		player.IsConnected = connected
		player.LastActivity = time.Now()
	}
	room.LastActivity = time.Now()
}

// RemovePlayer 真的把玩家移出房間（使用者主動離開才呼叫）
func (s *RoomService) RemovePlayer(roomID, playerID string) {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	room.RemovePlayer(playerID)
	room.LastActivity = time.Now()
}

// Touch 更新房間活動時間，避免使用中的房間被清理
func (s *RoomService) Touch(roomID string) {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return
	}

	s.mu.Lock()
	room.LastActivity = time.Now()
	s.mu.Unlock()
}

// UpdateSettings 大廳調整設定（題數 / 秒數 / 取名開關 / 重新編排題目）
func (s *RoomService) UpdateSettings(roomID string, timeLimit int, requireNickname *bool, questions []models.Question) error {
	room, err := s.GetRoom(roomID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if room.Status != models.RoomStatusWaiting {
		return errors.New("遊戲進行中無法修改設定")
	}

	if timeLimit >= 15 && timeLimit <= 300 {
		room.QuestionTimeLimit = timeLimit
	}
	if requireNickname != nil {
		room.RequireNickname = *requireNickname
	}
	if len(questions) > 0 {
		room.Questions = questions
	}

	room.LastActivity = time.Now()
	return nil
}

// DeleteRoom 刪除房間
func (s *RoomService) DeleteRoom(roomID string) {
	s.mu.Lock()
	delete(s.rooms, strings.ToUpper(roomID))
	s.mu.Unlock()
}

// ExpiredRooms 找出超過 idleLimit 沒有活動的房間 ID
func (s *RoomService) ExpiredRooms(idleLimit time.Duration) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	expired := make([]string, 0)
	for id, room := range s.rooms {
		if now.Sub(room.LastActivity) > idleLimit {
			expired = append(expired, id)
		}
	}
	return expired
}

// RoomCount 目前房間數
func (s *RoomService) RoomCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.rooms)
}

// generateRoomID 產生不重複的房號。去掉容易看錯的 0/O/1/I。
func (s *RoomService) generateRoomID() string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	length := s.cfg.Game.RoomIDLength
	if length <= 0 {
		length = 6
	}

	for attempt := 0; attempt < 100; attempt++ {
		b := make([]byte, length)
		for i := range b {
			b[i] = charset[rand.Intn(len(charset))]
		}
		id := string(b)

		s.mu.RLock()
		_, exists := s.rooms[id]
		s.mu.RUnlock()

		if !exists {
			return id
		}
	}

	return fmt.Sprintf("R%d", time.Now().UnixNano()%100000)
}
