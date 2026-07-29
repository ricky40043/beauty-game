package websocket_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"beauty-game/internal/config"
	"beauty-game/internal/handlers"
	"beauty-game/internal/models"
	"beauty-game/internal/services"
	"beauty-game/internal/storage"
	ws "beauty-game/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// fakeJPEG 造一張有合法 JPEG magic bytes 的假照片
func fakeJPEG(size int) []byte {
	data := make([]byte, size)
	copy(data, []byte{0xFF, 0xD8, 0xFF, 0xE0})
	return data
}

type testServer struct {
	http  *httptest.Server
	wsURL string
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := config.Load()
	cfg.Environment = "test"

	photoStore := storage.New(storage.Limits{
		MaxPhotoBytes: cfg.Photo.MaxPhotoBytes,
		MaxRoomPhotos: cfg.Photo.MaxRoomPhotos,
		MaxRoomBytes:  cfg.Photo.MaxRoomBytes,
	})
	questionService := services.NewQuestionService()
	gameService := services.NewGameService()
	roomService := services.NewRoomService(cfg, questionService)

	hub := ws.NewHub(roomService, gameService, questionService, photoStore, "")
	go hub.Run()

	api := handlers.NewAPIHandler(roomService, questionService, hub)
	photo := handlers.NewPhotoHandler(photoStore, roomService, cfg.Photo.MaxPhotoBytes)

	router := gin.New()
	router.GET("/ws", api.ServeWS)
	router.POST("/api/rooms/:roomId/photos", photo.Upload)
	router.GET("/api/photos/:photoId", photo.Get)

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &testServer{
		http:  srv,
		wsURL: "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws",
	}
}

// conn 一條測試用的客戶端連線，會把收到的訊息全部留著供斷言
type conn struct {
	t    *testing.T
	c    *websocket.Conn
	msgs chan ws.Message
}

func (s *testServer) dial(t *testing.T) *conn {
	t.Helper()

	c, _, err := websocket.DefaultDialer.Dial(s.wsURL, nil)
	if err != nil {
		t.Fatalf("連線失敗: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	cn := &conn{t: t, c: c, msgs: make(chan ws.Message, 256)}
	go func() {
		for {
			var msg ws.Message
			if err := c.ReadJSON(&msg); err != nil {
				close(cn.msgs)
				return
			}
			cn.msgs <- msg
		}
	}()

	return cn
}

func (c *conn) send(msgType string, data any) {
	c.t.Helper()
	if err := c.c.WriteJSON(ws.Message{Type: msgType, Data: data}); err != nil {
		c.t.Fatalf("送出 %s 失敗: %v", msgType, err)
	}
}

// await 等某個類型的訊息出現，並把 data 解回 map
func (c *conn) await(msgType string) map[string]any {
	c.t.Helper()

	deadline := time.After(5 * time.Second)
	for {
		select {
		case msg, ok := <-c.msgs:
			if !ok {
				c.t.Fatalf("等 %s 時連線已關閉", msgType)
			}
			if msg.Type == "ERROR" {
				c.t.Fatalf("等 %s 卻收到 ERROR: %v", msgType, msg.Data)
			}
			if msg.Type == msgType {
				out, _ := msg.Data.(map[string]any)
				return out
			}
		case <-deadline:
			c.t.Fatalf("等 %s 逾時", msgType)
		}
	}
}

// uploadPhoto 走真正的 HTTP multipart 上傳，回傳 photoId
func (s *testServer) uploadPhoto(t *testing.T, roomID string) string {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("photo", "shot.jpg")
	if err != nil {
		t.Fatalf("建立 multipart 失敗: %v", err)
	}
	if _, err := part.Write(fakeJPEG(2048)); err != nil {
		t.Fatalf("寫入照片失敗: %v", err)
	}
	_ = writer.Close()

	req, _ := http.NewRequest(http.MethodPost, s.http.URL+"/api/rooms/"+roomID+"/photos", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.http.Client().Do(req)
	if err != nil {
		t.Fatalf("上傳失敗: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("上傳回應碼 %d，預期 201", resp.StatusCode)
	}

	var out struct {
		PhotoID string `json:"photoId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("解析上傳回應失敗: %v", err)
	}
	return out.PhotoID
}

// TestSoloGameFlow 單人模式完整跑一輪：開房 → 免取名入場 → 兩題拍照評選 → 結算
func TestSoloGameFlow(t *testing.T) {
	srv := newTestServer(t)

	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName":          "主持人",
		"mode":              "solo",
		"totalQuestions":    2,
		"questionTimeLimit": 30,
		"questionMode":      "random",
	})

	created := host.await("ROOM_CREATED")
	roomID, _ := created["roomId"].(string)
	if roomID == "" {
		t.Fatal("沒有拿到房號")
	}
	if created["requireNickname"] != false {
		t.Fatal("預設應該是免取名模式")
	}

	// 免取名：不帶 playerName 也能進場，伺服器自動配暱稱
	players := make([]*conn, 3)
	for i := range players {
		players[i] = srv.dial(t)
		players[i].send("JOIN_ROOM", map[string]any{"roomId": roomID})

		joined := players[i].await("JOIN_SUCCESS")
		name, _ := joined["playerName"].(string)
		if name == "" {
			t.Fatalf("玩家 %d 沒有拿到自動暱稱", i)
		}
	}

	host.send("START_GAME", map[string]any{})
	host.await("GAME_STARTED")

	for round := 1; round <= 2; round++ {
		question := host.await("NEW_QUESTION")
		if text, _ := question["text"].(string); text == "" {
			t.Fatalf("第 %d 題沒有題目文字", round)
		}

		// 三位玩家各上傳一張，第三張送出後應該自動收桌
		photoIDs := make([]string, 0, 3)
		for i, p := range players {
			photoID := srv.uploadPhoto(t, roomID)
			p.send("SUBMIT_PHOTO", map[string]any{"photoId": photoID})

			accepted := p.await("PHOTO_ACCEPTED")
			if order, _ := accepted["order"].(float64); int(order) != i+1 {
				t.Fatalf("第 %d 位玩家的抵達順序應該是 %d，實際是 %v", i+1, i+1, accepted["order"])
			}
			photoIDs = append(photoIDs, photoID)
		}

		closed := host.await("ROUND_CLOSED")
		if total, _ := closed["totalPhotos"].(float64); int(total) != 3 {
			t.Fatalf("第 %d 題應該收到 3 張照片，實際 %v", round, closed["totalPhotos"])
		}

		// 房主挑前兩張當冠亞軍
		host.send("PICK_WINNERS", map[string]any{"photoIds": photoIDs[:2]})
		result := host.await("ROUND_RESULT")

		winners, _ := result["winners"].([]any)
		if len(winners) != 2 {
			t.Fatalf("第 %d 題應該有 2 位得獎，實際 %d", round, len(winners))
		}

		first, _ := winners[0].(map[string]any)
		if points, _ := first["points"].(float64); int(points) != 100 {
			t.Fatalf("冠軍應得 100 分，實際 %v", first["points"])
		}

		if round < 2 {
			host.send("NEXT_QUESTION", map[string]any{})
		}
	}

	host.send("NEXT_QUESTION", map[string]any{})
	finished := host.await("GAME_FINISHED")

	scores, _ := finished["scores"].([]any)
	if len(scores) != 3 {
		t.Fatalf("結算應該有 3 位玩家，實際 %d", len(scores))
	}

	top, _ := scores[0].(map[string]any)
	topScore, _ := top["score"].(float64)

	// 每題：參與分 10 + 速度分（15/10/5）+ 得獎分（100/80/0），兩題翻倍
	if int(topScore) != (10+15+100)*2 {
		t.Fatalf("第一名分數應為 %d，實際 %v", (10+15+100)*2, topScore)
	}
}

// TestSoloResubmitReplacesPhoto 單人模式重拍會覆蓋舊照片、沿用原本的抵達順序，
// 不會多算一張，也不會重複拿參與分；收桌之後則不再接受投稿。
func TestSoloResubmitReplacesPhoto(t *testing.T) {
	srv := newTestServer(t)

	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "solo",
		"totalQuestions": 1, "questionTimeLimit": 30, "questionMode": "random",
	})
	roomID, _ := host.await("ROOM_CREATED")["roomId"].(string)

	alice := srv.dial(t)
	alice.send("JOIN_ROOM", map[string]any{"roomId": roomID, "playerName": "小美"})
	aliceID, _ := alice.await("JOIN_SUCCESS")["playerId"].(string)

	bob := srv.dial(t)
	bob.send("JOIN_ROOM", map[string]any{"roomId": roomID, "playerName": "阿明"})
	bob.await("JOIN_SUCCESS")

	host.send("START_GAME", map[string]any{})
	host.await("GAME_STARTED")
	host.await("NEW_QUESTION")

	first := srv.uploadPhoto(t, roomID)
	alice.send("SUBMIT_PHOTO", map[string]any{"photoId": first})
	alice.await("PHOTO_ACCEPTED")

	// 小美重拍：順序仍是 1，且整場照片數維持 1 張
	second := srv.uploadPhoto(t, roomID)
	alice.send("SUBMIT_PHOTO", map[string]any{"photoId": second})

	replaced := alice.await("PHOTO_ACCEPTED")
	if ok, _ := replaced["replaced"].(bool); !ok {
		t.Fatal("重拍應該被標記為覆蓋")
	}
	if order, _ := replaced["order"].(float64); int(order) != 1 {
		t.Fatalf("重拍後抵達順序仍應是 1，實際 %v", replaced["order"])
	}

	broadcast := host.await("PHOTO_SUBMITTED")
	for {
		// 抓到覆蓋那一則廣播為止
		if isReplaced, _ := broadcast["replaced"].(bool); isReplaced {
			break
		}
		broadcast = host.await("PHOTO_SUBMITTED")
	}
	if submitted, _ := broadcast["submitted"].(float64); int(submitted) != 1 {
		t.Fatalf("覆蓋後投稿數應該還是 1，實際 %v", broadcast["submitted"])
	}

	// 舊照片要從記憶體釋放掉
	resp, err := srv.http.Client().Get(srv.http.URL + "/api/photos/" + first)
	if err != nil {
		t.Fatalf("查舊照片失敗: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("被覆蓋的舊照片應該已刪除，實際回應碼 %d", resp.StatusCode)
	}

	// 阿明交卷後全員到齊，自動收桌
	bobPhoto := srv.uploadPhoto(t, roomID)
	bob.send("SUBMIT_PHOTO", map[string]any{"photoId": bobPhoto})
	bob.await("PHOTO_ACCEPTED")

	closed := host.await("ROUND_CLOSED")
	if total, _ := closed["totalPhotos"].(float64); int(total) != 2 {
		t.Fatalf("收桌時應該有 2 張照片，實際 %v", closed["totalPhotos"])
	}

	// 收桌後就不該再收投稿
	late := srv.uploadPhoto(t, roomID)
	alice.send("SUBMIT_PHOTO", map[string]any{"photoId": late})

	deadline := time.After(3 * time.Second)
	for {
		select {
		case msg, ok := <-alice.msgs:
			if !ok {
				t.Fatal("連線意外關閉")
			}
			if msg.Type == "ERROR" {
				data, _ := msg.Data.(map[string]any)
				if code, _ := data["code"].(string); code != "NOT_SHOOTING" {
					t.Fatalf("預期 NOT_SHOOTING，實際 %v", code)
				}

				// 小美只拿一次參與分與速度分：10 + 15
				host.send("PICK_WINNERS", map[string]any{"photoIds": []string{second}})
				result := host.await("ROUND_RESULT")
				scores, _ := result["scores"].([]any)
				for _, raw := range scores {
					s, _ := raw.(map[string]any)
					if id, _ := s["playerId"].(string); id == aliceID {
						if score, _ := s["score"].(float64); int(score) != 10+15+100 {
							t.Fatalf("小美應得 %d 分，實際 %v", 10+15+100, s["score"])
						}
						return
					}
				}
				t.Fatal("排行榜裡找不到小美")
			}
			if msg.Type == "PHOTO_ACCEPTED" {
				t.Fatal("收桌後不應該再接受投稿")
			}
		case <-deadline:
			t.Fatal("等錯誤回覆逾時")
		}
	}
}

// TestGroupModeBonus 團體模式：得獎照片的拍攝者拿名次分，全場再一起拿合作分
func TestGroupModeBonus(t *testing.T) {
	srv := newTestServer(t)

	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "group",
		"totalQuestions": 1, "questionTimeLimit": 30, "questionMode": "random",
	})
	roomID, _ := host.await("ROOM_CREATED")["roomId"].(string)

	shooter := srv.dial(t)
	shooter.send("JOIN_ROOM", map[string]any{"roomId": roomID, "playerName": "攝影師"})
	shooterID, _ := shooter.await("JOIN_SUCCESS")["playerId"].(string)

	bystander := srv.dial(t)
	bystander.send("JOIN_ROOM", map[string]any{"roomId": roomID, "playerName": "路人"})
	bystanderID, _ := bystander.await("JOIN_SUCCESS")["playerId"].(string)

	host.send("START_GAME", map[string]any{})
	host.await("GAME_STARTED")
	host.await("NEW_QUESTION")

	photoID := srv.uploadPhoto(t, roomID)
	shooter.send("SUBMIT_PHOTO", map[string]any{"photoId": photoID})
	shooter.await("PHOTO_ACCEPTED")

	// 團體模式不會因為「全員交卷」自動收桌，由房主結束
	host.send("END_SHOOTING", map[string]any{})
	host.await("ROUND_CLOSED")

	host.send("PICK_WINNERS", map[string]any{"photoIds": []string{photoID}})
	result := host.await("ROUND_RESULT")

	if bonus, _ := result["groupBonus"].(float64); int(bonus) != services.PointsGroupBonus {
		t.Fatalf("團體模式應該有 %d 分合作分，實際 %v", services.PointsGroupBonus, result["groupBonus"])
	}

	byID := map[string]int{}
	scores, _ := result["scores"].([]any)
	for _, raw := range scores {
		s, _ := raw.(map[string]any)
		id, _ := s["playerId"].(string)
		score, _ := s["score"].(float64)
		byID[id] = int(score)
	}

	// 攝影師：參與 10 + 速度 15 + 冠軍 100 + 合作 50
	if got, want := byID[shooterID], 10+15+100+services.PointsGroupBonus; got != want {
		t.Fatalf("攝影師應得 %d 分，實際 %d", want, got)
	}
	// 沒拍照的人也拿得到合作分
	if got, want := byID[bystanderID], services.PointsGroupBonus; got != want {
		t.Fatalf("同隊玩家應得 %d 分，實際 %d", want, got)
	}
}

// TestCustomQuestionOrder 房主自選題目時，出題順序必須完全照他排的走
func TestCustomQuestionOrder(t *testing.T) {
	srv := newTestServer(t)

	bank := services.AllQuestions(models.ModeSolo)
	if len(bank) < 3 {
		t.Fatal("題庫太小")
	}
	picked := []int{bank[4].ID, bank[0].ID, bank[2].ID}

	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "solo",
		"questionTimeLimit": 30,
		"questionMode":      "custom",
		"questionIds":       picked,
		"customQuestions":   []string{"擺出你最帥的姿勢"},
	})

	created := host.await("ROOM_CREATED")
	roomID, _ := created["roomId"].(string)
	if total, _ := created["totalQuestions"].(float64); int(total) != 4 {
		t.Fatalf("應該有 4 題（3 題庫 + 1 自訂），實際 %v", created["totalQuestions"])
	}

	player := srv.dial(t)
	player.send("JOIN_ROOM", map[string]any{"roomId": roomID})
	player.await("JOIN_SUCCESS")

	host.send("START_GAME", map[string]any{})
	host.await("GAME_STARTED")

	expected := []string{bank[4].Text, bank[0].Text, bank[2].Text, "擺出你最帥的姿勢"}
	for i, want := range expected {
		question := host.await("NEW_QUESTION")
		if got, _ := question["text"].(string); got != want {
			t.Fatalf("第 %d 題應該是「%s」，實際是「%s」", i+1, want, got)
		}

		host.send("END_SHOOTING", map[string]any{})
		host.await("ROUND_CLOSED")
		host.await("ROUND_RESULT") // 沒人投稿，自動空結果
		host.send("NEXT_QUESTION", map[string]any{})
	}

	host.await("GAME_FINISHED")
}

// TestRejoinRestoresState 重連後房主與玩家都能拿回完整狀態
func TestRejoinRestoresState(t *testing.T) {
	srv := newTestServer(t)

	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "solo",
		"totalQuestions": 2, "questionTimeLimit": 60, "questionMode": "random",
	})
	created := host.await("ROOM_CREATED")
	roomID, _ := created["roomId"].(string)
	hostToken, _ := created["hostToken"].(string)

	player := srv.dial(t)
	player.send("JOIN_ROOM", map[string]any{"roomId": roomID, "playerName": "小花"})
	playerID, _ := player.await("JOIN_SUCCESS")["playerId"].(string)

	host.send("START_GAME", map[string]any{})
	host.await("GAME_STARTED")
	host.await("NEW_QUESTION")

	// 玩家換一條連線重連
	player2 := srv.dial(t)
	player2.send("REJOIN_ROOM", map[string]any{"roomId": roomID, "playerId": playerID})
	restored := player2.await("REJOIN_SUCCESS")

	if status, _ := restored["status"].(string); status != string(models.RoomStatusShooting) {
		t.Fatalf("重連後狀態應該是 shooting，實際 %v", restored["status"])
	}
	if restored["question"] == nil {
		t.Fatal("重連後應該拿得到當前題目")
	}

	// 房主用 hostToken 重連，換連線後仍是房主
	host2 := srv.dial(t)
	host2.send("REJOIN_ROOM", map[string]any{"roomId": roomID, "hostToken": hostToken})
	hostRestored := host2.await("REJOIN_SUCCESS")

	if isHost, _ := hostRestored["isHost"].(bool); !isHost {
		t.Fatal("帶 hostToken 重連應該還是房主")
	}

	// 新的房主連線可以繼續控場
	host2.send("END_SHOOTING", map[string]any{})
	host2.await("ROUND_CLOSED")
}

// TestPhotoServedBack 上傳的照片可以透過 /api/photos/:id 取回
func TestPhotoServedBack(t *testing.T) {
	srv := newTestServer(t)

	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "solo",
		"totalQuestions": 1, "questionTimeLimit": 30, "questionMode": "random",
	})
	roomID, _ := host.await("ROOM_CREATED")["roomId"].(string)

	photoID := srv.uploadPhoto(t, roomID)

	resp, err := srv.http.Client().Get(srv.http.URL + "/api/photos/" + photoID)
	if err != nil {
		t.Fatalf("取照片失敗: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("取照片回應碼 %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "image/jpeg") {
		t.Fatalf("Content-Type 應該是 image/jpeg，實際 %s", ct)
	}
}

// TestRejectsNonImageUpload 非圖片內容要被擋下來
func TestRejectsNonImageUpload(t *testing.T) {
	srv := newTestServer(t)

	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "solo",
		"totalQuestions": 1, "questionTimeLimit": 30, "questionMode": "random",
	})
	roomID, _ := host.await("ROOM_CREATED")["roomId"].(string)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("photo", "evil.jpg")
	_, _ = part.Write([]byte("<html>this is not a photo</html>"))
	_ = writer.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.http.URL+"/api/rooms/"+roomID+"/photos", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := srv.http.Client().Do(req)
	if err != nil {
		t.Fatalf("請求失敗: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		t.Fatal("非圖片內容不應該被接受")
	}
}

// TestUploadToUnknownRoomRejected 不存在的房間不能當上傳目標
func TestUploadToUnknownRoomRejected(t *testing.T) {
	srv := newTestServer(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("photo", "shot.jpg")
	_, _ = part.Write(fakeJPEG(1024))
	_ = writer.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.http.URL+"/api/rooms/ZZZZZZ/photos", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := srv.http.Client().Do(req)
	if err != nil {
		t.Fatalf("請求失敗: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("應該回 404，實際 %d", resp.StatusCode)
	}
}

// TestNicknameRequiredMode 開啟自行取名時，沒帶名字要被擋下
func TestNicknameRequiredMode(t *testing.T) {
	srv := newTestServer(t)

	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "solo",
		"totalQuestions": 1, "questionTimeLimit": 30, "questionMode": "random",
		"requireNickname": true,
	})
	roomID, _ := host.await("ROOM_CREATED")["roomId"].(string)

	player := srv.dial(t)
	player.send("JOIN_ROOM", map[string]any{"roomId": roomID})

	deadline := time.After(3 * time.Second)
	for {
		select {
		case msg, ok := <-player.msgs:
			if !ok {
				t.Fatal("連線意外關閉")
			}
			if msg.Type == "JOIN_SUCCESS" {
				t.Fatal("要求取名的房間不該讓空名字進場")
			}
			if msg.Type == "ERROR" {
				data, _ := msg.Data.(map[string]any)
				if code, _ := data["code"].(string); code != "NAME_REQUIRED" {
					t.Fatalf("預期 NAME_REQUIRED，實際 %v", code)
				}
				return
			}
		case <-deadline:
			t.Fatal("等錯誤回覆逾時")
		}
	}
}

// TestTimerClosesRound 倒數歸零會自動收桌
func TestTimerClosesRound(t *testing.T) {
	srv := newTestServer(t)

	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "group",
		"totalQuestions": 1,
		// 低於 15 秒會被伺服器改回預設值，這裡用最小合法值讓測試等 15 秒內結束
		"questionTimeLimit": 15,
		"questionMode":      "random",
	})
	roomID, _ := host.await("ROOM_CREATED")["roomId"].(string)

	player := srv.dial(t)
	player.send("JOIN_ROOM", map[string]any{"roomId": roomID})
	player.await("JOIN_SUCCESS")

	host.send("START_GAME", map[string]any{})
	host.await("GAME_STARTED")
	host.await("NEW_QUESTION")

	tick := host.await("TIMER_UPDATE")
	if left, _ := tick["timeLeft"].(float64); int(left) != 14 {
		t.Fatalf("第一次倒數應該是 14，實際 %v", tick["timeLeft"])
	}

	// 等倒數自然歸零（15 秒上限再加一點緩衝）
	deadline := time.After(20 * time.Second)
	for {
		select {
		case msg, ok := <-host.msgs:
			if !ok {
				t.Fatal("連線意外關閉")
			}
			if msg.Type == "ROUND_CLOSED" {
				data, _ := msg.Data.(map[string]any)
				if reason, _ := data["reason"].(string); !strings.Contains(reason, "時間到") {
					t.Fatalf("收桌原因應該是時間到，實際 %v", reason)
				}
				return
			}
		case <-deadline:
			t.Fatal("等倒數收桌逾時")
		}
	}
}
