package services_test

import (
	"fmt"
	"testing"

	"beauty-game/internal/models"
	"beauty-game/internal/services"
)

// judgingRoom 造一間正在評選、有 n 張投稿的房間
func judgingRoom(n, maxWinners int) (*models.Room, []string) {
	room := &models.Room{
		Mode:            models.ModeSolo,
		Status:          models.RoomStatusJudging,
		Players:         make(map[string]*models.Player),
		Questions:       []models.Question{{ID: 1001, Text: "測試題"}},
		CurrentQuestion: 0,
		MaxWinners:      maxWinners,
	}
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		pid := fmt.Sprintf("p%d", i)
		room.Players[pid] = &models.Player{ID: pid, Name: pid}
		photoID := fmt.Sprintf("photo%d", i)
		room.RoundPhotos = append(room.RoundPhotos, &models.PhotoSubmission{PhotoID: photoID, PlayerID: pid})
		ids = append(ids, photoID)
	}
	return room, ids
}

// TestApplyWinnersUnlimited 名額設成不限時，全部投稿都能排名，第 6 名以後各拿 10 分
func TestApplyWinnersUnlimited(t *testing.T) {
	room, ids := judgingRoom(8, 0)
	result, _, err := services.NewGameService().ApplyWinners(room, ids)
	if err != nil {
		t.Fatalf("不限名額時選 8 張應該成功：%v", err)
	}
	want := []int{100, 80, 60, 40, 20, 10, 10, 10}
	for i, w := range result.Winners {
		if w.Rank != i+1 || w.Points != want[i] {
			t.Fatalf("第 %d 名應該是 %d 分，實際 rank=%d points=%d", i+1, want[i], w.Rank, w.Points)
		}
	}
}

// TestApplyWinnersRespectsLimit 超過房間設定的名額要擋下來
func TestApplyWinnersRespectsLimit(t *testing.T) {
	room, ids := judgingRoom(4, 2)
	if _, _, err := services.NewGameService().ApplyWinners(room, ids[:3]); err == nil {
		t.Fatal("名額 2 卻選了 3 張，應該被拒絕")
	}
	if _, _, err := services.NewGameService().ApplyWinners(room, ids[:2]); err != nil {
		t.Fatalf("名額內應該成功：%v", err)
	}
}

// TestLeaderboardSharesRankOnTie 同分同名次，下一名跳號
func TestLeaderboardSharesRankOnTie(t *testing.T) {
	room := &models.Room{Players: map[string]*models.Player{
		"a": {ID: "a", Name: "a", Score: 300},
		"b": {ID: "b", Name: "b", Score: 300},
		"c": {ID: "c", Name: "c", Score: 100},
	}}
	got := room.Leaderboard()
	if got[0].Rank != 1 || got[1].Rank != 1 || got[2].Rank != 3 {
		t.Fatalf("名次應該是 1、1、3，實際 %d、%d、%d", got[0].Rank, got[1].Rank, got[2].Rank)
	}
}
