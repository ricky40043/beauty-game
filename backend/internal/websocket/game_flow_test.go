package websocket_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
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
	http   *httptest.Server
	wsURL  string
	photos *storage.PhotoStore
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
	questionStore, err := services.NewQuestionStore(t.TempDir())
	if err != nil {
		t.Fatalf("建立題目儲存失敗: %v", err)
	}
	questionService := services.NewQuestionService(questionStore)
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
	router.GET("/api/rooms/:roomId/winners.zip", photo.WinnersZip)

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &testServer{
		http:   srv,
		wsURL:  "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws",
		photos: photoStore,
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
		// 沒人投稿：自動送空結果並直接進下一題，不用房主再按「下一題」
		if result := host.await("ROUND_RESULT"); result["empty"] != true {
			t.Fatalf("第 %d 題沒人投稿，應該是空結果", i+1)
		}
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

// TestGroupModeOneUploadPerPlayer 團體模式同一個人也只能有一張投稿。
// 之前允許一人三張，會讓單一玩家洗版、還把「已交卷 3/1」這種數字弄得沒意義。
func TestGroupModeOneUploadPerPlayer(t *testing.T) {
	srv := newTestServer(t)

	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "group",
		"totalQuestions": 1, "questionTimeLimit": 60, "questionMode": "random",
	})
	roomID, _ := host.await("ROOM_CREATED")["roomId"].(string)

	player := srv.dial(t)
	player.send("JOIN_ROOM", map[string]any{"roomId": roomID, "playerName": "阿美"})
	player.await("JOIN_SUCCESS")

	host.send("START_GAME", map[string]any{})
	host.await("GAME_STARTED")
	host.await("NEW_QUESTION")

	// 連傳三張，全部由同一個人送出
	for i := 1; i <= 3; i++ {
		player.send("SUBMIT_PHOTO", map[string]any{"photoId": srv.uploadPhoto(t, roomID)})
		accepted := player.await("PHOTO_ACCEPTED")

		if order, _ := accepted["order"].(float64); int(order) != 1 {
			t.Fatalf("第 %d 次上傳的順序應該都是 1，實際 %v", i, accepted["order"])
		}
		if replaced, _ := accepted["replaced"].(bool); replaced != (i > 1) {
			t.Fatalf("第 %d 次上傳的 replaced 應為 %v，實際 %v", i, i > 1, replaced)
		}
	}

	host.send("END_SHOOTING", map[string]any{})
	closed := host.await("ROUND_CLOSED")

	if total, _ := closed["totalPhotos"].(float64); int(total) != 1 {
		t.Fatalf("一個人連傳三次，房內仍應只有 1 張照片，實際 %v", closed["totalPhotos"])
	}

	// 參與分與速度分也只能拿一次
	photos, _ := closed["photos"].([]any)
	first, _ := photos[0].(map[string]any)
	host.send("PICK_WINNERS", map[string]any{"photoIds": []string{first["photoId"].(string)}})
	result := host.await("ROUND_RESULT")

	scores, _ := result["scores"].([]any)
	top, _ := scores[0].(map[string]any)
	want := 10 + 15 + 100 + services.PointsGroupBonus
	if got, _ := top["score"].(float64); int(got) != want {
		t.Fatalf("分數應為 %d（參與+速度各一次），實際 %v", want, top["score"])
	}
}

// TestPracticeRound 試玩回合：不計分、不倒數，由房主結束後才進正式第 1 題
func TestPracticeRound(t *testing.T) {
	srv := newTestServer(t)

	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "solo",
		"totalQuestions": 2, "questionTimeLimit": 60, "questionMode": "random",
		"practiceRound": true,
	})
	created := host.await("ROOM_CREATED")
	roomID, _ := created["roomId"].(string)
	if enabled, _ := created["practiceEnabled"].(bool); !enabled {
		t.Fatal("建房時 practiceRound 沒有生效")
	}

	player := srv.dial(t)
	player.send("JOIN_ROOM", map[string]any{"roomId": roomID, "playerName": "小新"})
	playerID, _ := player.await("JOIN_SUCCESS")["playerId"].(string)

	host.send("START_GAME", map[string]any{})
	host.await("GAME_STARTED")

	practice := host.await("NEW_QUESTION")
	if isPractice, _ := practice["isPractice"].(bool); !isPractice {
		t.Fatal("第一個回合應該是試玩")
	}
	if left, _ := practice["timeLeft"].(float64); int(left) != 0 {
		t.Fatalf("試玩不該倒數，timeLeft 應為 0，實際 %v", practice["timeLeft"])
	}

	// 試玩投稿不計分
	player.send("SUBMIT_PHOTO", map[string]any{"photoId": srv.uploadPhoto(t, roomID)})
	player.await("PHOTO_ACCEPTED")

	// 只有一位玩家，若誤用正式規則會在這裡自動收桌 —— 試玩不該收桌
	select {
	case msg := <-host.msgs:
		if msg.Type == "ROUND_CLOSED" {
			t.Fatal("試玩回合不該自動收桌")
		}
	case <-time.After(1500 * time.Millisecond):
	}

	host.send("END_PRACTICE", map[string]any{})
	host.await("PRACTICE_ENDED")

	real := host.await("NEW_QUESTION")
	if isPractice, _ := real["isPractice"].(bool); isPractice {
		t.Fatal("結束試玩後應該進正式題")
	}
	if num, _ := real["questionNum"].(float64); int(num) != 1 {
		t.Fatalf("正式第一題的題號應為 1，實際 %v", real["questionNum"])
	}

	// 試玩拿到的分數不算數，正式開賽時應該還是 0
	host.send("END_SHOOTING", map[string]any{})
	host.await("ROUND_CLOSED")
	host.send("SKIP_ROUND", map[string]any{})
	scores, _ := host.await("ROUND_RESULT")["scores"].([]any)

	for _, raw := range scores {
		s, _ := raw.(map[string]any)
		if id, _ := s["playerId"].(string); id == playerID {
			if score, _ := s["score"].(float64); score != 0 {
				t.Fatalf("試玩不該計分，實際拿到 %v 分", score)
			}
			return
		}
	}
	t.Fatal("排行榜裡找不到玩家")
}

// TestPhotosFreedBetweenRounds 換題時要釋放上一題的非得獎照片。
//
// 沒有這個機制的話，整場遊戲的照片會一路堆在記憶體裡，把每間房的配額
// （預設 300 張 / 80MB）吃光 —— 20 人玩到第 15 題左右，玩家上傳就會開始失敗。
// 得獎照片必須留著，結算頁的照片牆靠它們。
func TestPhotosFreedBetweenRounds(t *testing.T) {
	srv := newTestServer(t)

	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "solo",
		"totalQuestions": 3, "questionTimeLimit": 60, "questionMode": "random",
	})
	roomID, _ := host.await("ROOM_CREATED")["roomId"].(string)

	players := make([]*conn, 3)
	for i := range players {
		players[i] = srv.dial(t)
		players[i].send("JOIN_ROOM", map[string]any{"roomId": roomID})
		players[i].await("JOIN_SUCCESS")
	}

	host.send("START_GAME", map[string]any{})
	host.await("GAME_STARTED")

	winnerPhotoIDs := make([]string, 0, 3)
	host.await("NEW_QUESTION")

	for round := 1; round <= 3; round++ {
		roundPhotos := make([]string, 0, 3)
		for _, p := range players {
			photoID := srv.uploadPhoto(t, roomID)
			p.send("SUBMIT_PHOTO", map[string]any{"photoId": photoID})
			p.await("PHOTO_ACCEPTED")
			roundPhotos = append(roundPhotos, photoID)
		}
		host.await("ROUND_CLOSED")

		// 每題只選一張得獎
		host.send("PICK_WINNERS", map[string]any{"photoIds": roundPhotos[:1]})
		host.await("ROUND_RESULT")
		winnerPhotoIDs = append(winnerPhotoIDs, roundPhotos[0])

		host.send("NEXT_QUESTION", map[string]any{})

		// 等伺服器真的推進到下一題（或收場）再斷言，否則會跟非同步處理搶跑
		if round < 3 {
			host.await("NEW_QUESTION")
		} else {
			host.await("GAME_FINISHED")
		}

		// 每題 3 張、只有 1 張得獎，所以做完第 N 題應該只剩 N 張
		if got, _, _ := srv.photos.Stats(); got != round {
			t.Fatalf("第 %d 題結束後照片庫應只剩 %d 張得獎照片，實際 %d 張", round, round, got)
		}
	}

	// 結算頁的照片牆要讀得到每一張得獎照片
	for i, photoID := range winnerPhotoIDs {
		resp, err := srv.http.Client().Get(srv.http.URL + "/api/photos/" + photoID)
		if err != nil {
			t.Fatalf("取第 %d 題得獎照片失敗: %v", i+1, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("第 %d 題的得獎照片被誤刪了，狀態碼 %d", i+1, resp.StatusCode)
		}
	}

	// 總共上傳 9 張，只有 3 張得獎，其餘 6 張應該都釋放掉了
	if got, _, _ := srv.photos.Stats(); got != 3 {
		t.Fatalf("整場結束後應只剩 3 張得獎照片，實際 %d 張", got)
	}
}

// TestResultAutoAdvance 公布得獎後停 resultSeconds 秒自動進下一題；
// 房主在倒數中按「下一題」也只會前進一題，不會跟計時器各跳一次
func TestResultAutoAdvance(t *testing.T) {
	srv := newTestServer(t)

	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "solo",
		"totalQuestions": 3, "questionTimeLimit": 60, "questionMode": "random",
		"resultSeconds": 1,
	})
	roomID, _ := host.await("ROOM_CREATED")["roomId"].(string)

	player := srv.dial(t)
	player.send("JOIN_ROOM", map[string]any{"roomId": roomID})
	player.await("JOIN_SUCCESS")

	host.send("START_GAME", map[string]any{})
	host.await("NEW_QUESTION")

	pickOnly := func() {
		photoID := srv.uploadPhoto(t, roomID)
		player.send("SUBMIT_PHOTO", map[string]any{"photoId": photoID})
		host.await("ROUND_CLOSED")
		host.send("PICK_WINNERS", map[string]any{"photoIds": []string{photoID}})
		if sec, _ := host.await("ROUND_RESULT")["autoNextSec"].(float64); sec != 1 {
			t.Fatalf("ROUND_RESULT 應該帶 autoNextSec=1，實際 %v", sec)
		}
	}

	// 第 1 題：不按任何鈕，1 秒後應該自己進第 2 題
	pickOnly()
	if num, _ := host.await("NEW_QUESTION")["questionNum"].(float64); num != 2 {
		t.Fatalf("自動換題應該進第 2 題，實際 %v", num)
	}

	// 第 2 題：公布後馬上按「下一題」，計時器到點時不能再多跳一題
	pickOnly()
	host.send("NEXT_QUESTION", map[string]any{})
	if num, _ := host.await("NEW_QUESTION")["questionNum"].(float64); num != 3 {
		t.Fatalf("按下一題應該進第 3 題，實際 %v", num)
	}

	deadline := time.After(1600 * time.Millisecond)
	for {
		select {
		case msg := <-host.msgs:
			if msg.Type == "NEW_QUESTION" || msg.Type == "GAME_FINISHED" {
				t.Fatalf("計時器在房主已換題後又換了一次：收到 %s", msg.Type)
			}
		case <-deadline:
			return
		}
	}
}

// TestWinnersZip 結算後可以把每題第 1 名的照片打包下載；沒選得獎的題目不會出現在 zip 裡
func TestWinnersZip(t *testing.T) {
	srv := newTestServer(t)

	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "solo",
		"totalQuestions": 3, "questionTimeLimit": 60, "questionMode": "random",
		"resultSeconds": 0,
	})
	roomID, _ := host.await("ROOM_CREATED")["roomId"].(string)

	alice := srv.dial(t)
	alice.send("JOIN_ROOM", map[string]any{"roomId": roomID, "playerName": "阿明"})
	alice.await("JOIN_SUCCESS")

	host.send("START_GAME", map[string]any{})
	host.await("NEW_QUESTION")

	for round := 1; round <= 3; round++ {
		photoID := srv.uploadPhoto(t, roomID)
		alice.send("SUBMIT_PHOTO", map[string]any{"photoId": photoID})
		host.await("ROUND_CLOSED")

		if round == 2 {
			// 第 2 題不選，伺服器會直接進下一題
			host.send("SKIP_ROUND", map[string]any{})
			host.await("NEW_QUESTION")
			continue
		}
		host.send("PICK_WINNERS", map[string]any{"photoIds": []string{photoID}})
		host.await("ROUND_RESULT")
		host.send("NEXT_QUESTION", map[string]any{})
		if round < 3 {
			host.await("NEW_QUESTION")
		}
	}
	host.await("GAME_FINISHED")

	resp, err := srv.http.Client().Get(srv.http.URL + "/api/rooms/" + roomID + "/winners.zip")
	if err != nil {
		t.Fatalf("下載 zip 失敗: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "application/zip" {
		t.Fatalf("zip 回應不對：%d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	raw, _ := io.ReadAll(resp.Body)
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatalf("不是合法的 zip: %v", err)
	}
	if len(zr.File) != 2 {
		t.Fatalf("3 題裡有 2 題有得獎，zip 應該有 2 張，實際 %d", len(zr.File))
	}
	for i, prefix := range []string{"第01題-", "第03題-"} {
		name := zr.File[i].Name
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, "-阿明.jpg") {
			t.Fatalf("第 %d 個檔名不對：%s", i+1, name)
		}
		f, _ := zr.File[i].Open()
		data, _ := io.ReadAll(f)
		f.Close()
		if !bytes.Equal(data, fakeJPEG(2048)) {
			t.Fatalf("%s 的內容跟上傳的照片不同", name)
		}
	}
}

// TestResetDrawsFreshQuestions 隨機出題的房間玩完「再來一局」要換一批沒出過的題目；
// 自選題目的房間則維持主持人排好的題目
func TestResetDrawsFreshQuestions(t *testing.T) {
	srv := newTestServer(t)

	questionIDs := func(raw any) []int {
		list, _ := raw.([]any)
		out := make([]int, 0, len(list))
		for _, item := range list {
			q, _ := item.(map[string]any)
			id, _ := q["id"].(float64)
			out = append(out, int(id))
		}
		return out
	}

	playThrough := func(host *conn, total int) {
		host.send("START_GAME", map[string]any{})
		for i := 0; i < total; i++ {
			host.await("NEW_QUESTION")
			host.send("END_SHOOTING", map[string]any{})
			host.await("ROUND_CLOSED")
		}
		host.await("GAME_FINISHED")
	}

	// ── 隨機：勾學生篇（11 題），每局 5 題，連玩兩局不能重複
	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "solo", "questionMode": "random",
		"totalQuestions": 5, "questionTimeLimit": 60, "categories": []string{"student"},
	})
	created := host.await("ROOM_CREATED")
	roomID, _ := created["roomId"].(string)
	first := questionIDs(created["questions"])

	player := srv.dial(t)
	player.send("JOIN_ROOM", map[string]any{"roomId": roomID})
	player.await("JOIN_SUCCESS")

	playThrough(host, 5)
	host.send("RESET_ROOM_TO_LOBBY", map[string]any{})
	second := questionIDs(host.await("ROOM_RESET_TO_LOBBY")["questions"])

	if len(second) != 5 {
		t.Fatalf("第二局應該還是 5 題，實際 %d", len(second))
	}
	seen := map[int]bool{}
	for _, id := range first {
		seen[id] = true
	}
	for _, id := range second {
		if seen[id] {
			t.Fatalf("第二局抽到第一局玩過的題目 %d（第一局 %v，第二局 %v）", id, first, second)
		}
	}

	// ── 自選：再來一局維持同樣的題目與順序
	host2 := srv.dial(t)
	bank := services.AllQuestions(models.ModeSolo)
	picked := []int{bank[3].ID, bank[1].ID}
	host2.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "solo", "questionMode": "custom",
		"questionIds": picked, "questionTimeLimit": 60,
	})
	room2, _ := host2.await("ROOM_CREATED")["roomId"].(string)
	p2 := srv.dial(t)
	p2.send("JOIN_ROOM", map[string]any{"roomId": room2})
	p2.await("JOIN_SUCCESS")

	playThrough(host2, 2)
	host2.send("RESET_ROOM_TO_LOBBY", map[string]any{})
	again := questionIDs(host2.await("ROOM_RESET_TO_LOBBY")["questions"])
	if len(again) != 2 || again[0] != picked[0] || again[1] != picked[1] {
		t.Fatalf("自選題目再來一局應維持 %v，實際 %v", picked, again)
	}
}

// TestVoteMode 投票模式：拍照中就能投、湊滿候選張數開始投票倒數、不能投自己、
// 可以改投，倒數結束依票數排名給分
func TestVoteMode(t *testing.T) {
	srv := newTestServer(t)

	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "solo", "questionMode": "random",
		"totalQuestions": 1, "questionTimeLimit": 60,
		"voteMode": true, "voteCandidates": 2, "voteSeconds": 5, "resultSeconds": 0,
	})
	created := host.await("ROOM_CREATED")
	roomID, _ := created["roomId"].(string)
	if created["voteMode"] != true {
		t.Fatal("建房回應應該帶 voteMode=true")
	}

	join := func(name string) (*conn, string) {
		p := srv.dial(t)
		p.send("JOIN_ROOM", map[string]any{"roomId": roomID, "playerName": name})
		id, _ := p.await("JOIN_SUCCESS")["playerId"].(string)
		return p, id
	}
	a, aID := join("阿明")
	b, bID := join("小華")
	c, _ := join("小美")

	host.send("START_GAME", map[string]any{})
	host.await("NEW_QUESTION")

	upload := func(p *conn) {
		photoID := srv.uploadPhoto(t, roomID)
		p.send("SUBMIT_PHOTO", map[string]any{"photoId": photoID})
	}

	upload(a)
	a.await("PHOTO_ACCEPTED")

	// 拍照中就能投；先投給阿明，等等改投
	b.send("CAST_VOTE", map[string]any{"targetPlayerId": aID})
	b.await("VOTE_ACCEPTED")

	// 不能投自己
	a.send("CAST_VOTE", map[string]any{"targetPlayerId": aID})
	if msg := waitFor(t, a, "ERROR", 3*time.Second); msg["code"] != "VOTE_FAILED" {
		t.Fatalf("投自己應該被拒絕，實際 %v", msg)
	}

	// 第 2 張湊滿候選，進投票倒數
	upload(b)
	started := host.await("VOTING_STARTED")
	if photos, _ := started["photos"].([]any); len(photos) != 2 {
		t.Fatalf("候選應該是 2 張，實際 %v", started["photos"])
	}

	// 湊滿後第 3 個人上傳要被擋
	upload(c)
	if msg := waitFor(t, c, "ERROR", 3*time.Second); msg["code"] != "NOT_SHOOTING" {
		t.Fatalf("候選滿了還能上傳：%v", msg)
	}

	a.send("CAST_VOTE", map[string]any{"targetPlayerId": bID})
	a.await("VOTE_ACCEPTED")
	c.send("CAST_VOTE", map[string]any{"targetPlayerId": bID})
	c.await("VOTE_ACCEPTED")
	// 小華改投：還是只能投阿明（不能投自己），重送一次確認改投不會重複計票
	b.send("CAST_VOTE", map[string]any{"targetPlayerId": aID})
	b.await("VOTE_ACCEPTED")

	// 投票倒數 5 秒，比 await 的 5 秒逾時還緊，另外等久一點
	result := waitFor(t, host, "ROUND_RESULT", 9*time.Second)
	winners, _ := result["winners"].([]any)
	if len(winners) != 2 {
		t.Fatalf("2 張都有票，應該 2 位得獎，實際 %d", len(winners))
	}
	first, _ := winners[0].(map[string]any)
	second, _ := winners[1].(map[string]any)
	if first["playerId"] != bID || first["votes"] != float64(2) || first["points"] != float64(100) {
		t.Fatalf("第 1 名應該是小華 2 票 100 分，實際 %v", first)
	}
	if second["playerId"] != aID || second["votes"] != float64(1) || second["rank"] != float64(2) {
		t.Fatalf("第 2 名應該是阿明 1 票，實際 %v", second)
	}
}

// waitFor 跟 await 一樣等某類訊息，但可以等 ERROR 本身、也能自訂逾時
func waitFor(t *testing.T, c *conn, msgType string, timeout time.Duration) map[string]any {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case msg, ok := <-c.msgs:
			if !ok {
				t.Fatalf("等 %s 時連線已關閉", msgType)
			}
			if msg.Type == msgType {
				out, _ := msg.Data.(map[string]any)
				return out
			}
			if msg.Type == "ERROR" {
				t.Fatalf("等 %s 卻收到 ERROR: %v", msgType, msg.Data)
			}
		case <-deadline:
			t.Fatalf("等 %s 逾時", msgType)
		}
	}
}

// TestNoUploadAfterCloseAndSkipWhileShooting 房主收桌（點照片評分）後其他人不能再上傳；
// 拍照中也能直接跳過這一題
func TestNoUploadAfterCloseAndSkipWhileShooting(t *testing.T) {
	srv := newTestServer(t)

	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "solo", "questionMode": "random",
		"totalQuestions": 3, "questionTimeLimit": 60,
	})
	roomID, _ := host.await("ROOM_CREATED")["roomId"].(string)

	a := srv.dial(t)
	a.send("JOIN_ROOM", map[string]any{"roomId": roomID})
	a.await("JOIN_SUCCESS")
	b := srv.dial(t)
	b.send("JOIN_ROOM", map[string]any{"roomId": roomID})
	b.await("JOIN_SUCCESS")

	host.send("START_GAME", map[string]any{})
	host.await("NEW_QUESTION")

	// 第 1 題：A 交了，房主點照片（END_SHOOTING）→ B 再交要被擋
	a.send("SUBMIT_PHOTO", map[string]any{"photoId": srv.uploadPhoto(t, roomID)})
	a.await("PHOTO_ACCEPTED")
	host.send("END_SHOOTING", map[string]any{})
	host.await("ROUND_CLOSED")
	b.send("SUBMIT_PHOTO", map[string]any{"photoId": srv.uploadPhoto(t, roomID)})
	if msg := waitFor(t, b, "ERROR", 3*time.Second); msg["code"] != "NOT_SHOOTING" {
		t.Fatalf("收桌後上傳應該被擋，實際 %v", msg)
	}
	host.send("SKIP_ROUND", map[string]any{})
	if num, _ := host.await("NEW_QUESTION")["questionNum"].(float64); num != 2 {
		t.Fatalf("評選中跳過應該進第 2 題，實際 %v", num)
	}

	// 第 2 題：拍照中直接跳過，應該馬上到第 3 題，舊的拍照倒數不能再收桌
	host.send("SKIP_ROUND", map[string]any{})
	if num, _ := host.await("NEW_QUESTION")["questionNum"].(float64); num != 3 {
		t.Fatalf("拍照中跳過應該進第 3 題，實際 %v", num)
	}
	a.send("SUBMIT_PHOTO", map[string]any{"photoId": srv.uploadPhoto(t, roomID)})
	if accepted := a.await("PHOTO_ACCEPTED"); accepted["order"] != float64(1) {
		t.Fatalf("第 3 題的第一張應該是第 1 個交，實際 %v", accepted["order"])
	}
}

// TestVoteTieGoesToEarlierUpload 投票同票時，先上傳的人名次較前、分數較多
func TestVoteTieGoesToEarlierUpload(t *testing.T) {
	srv := newTestServer(t)

	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "solo", "questionMode": "random",
		"totalQuestions": 1, "questionTimeLimit": 60,
		"voteMode": true, "voteCandidates": 2, "voteSeconds": 60, "resultSeconds": 0,
	})
	roomID, _ := host.await("ROOM_CREATED")["roomId"].(string)

	join := func() (*conn, string) {
		p := srv.dial(t)
		p.send("JOIN_ROOM", map[string]any{"roomId": roomID})
		id, _ := p.await("JOIN_SUCCESS")["playerId"].(string)
		return p, id
	}
	first, firstID := join()
	second, secondID := join()

	host.send("START_GAME", map[string]any{})
	host.await("NEW_QUESTION")

	first.send("SUBMIT_PHOTO", map[string]any{"photoId": srv.uploadPhoto(t, roomID)})
	first.await("PHOTO_ACCEPTED")
	second.send("SUBMIT_PHOTO", map[string]any{"photoId": srv.uploadPhoto(t, roomID)})
	started := host.await("VOTING_STARTED")
	if started["timeLeft"] != float64(60) {
		t.Fatalf("投票倒數應支援設到 60 秒，實際 %v", started["timeLeft"])
	}

	// 互投，各 1 票
	first.send("CAST_VOTE", map[string]any{"targetPlayerId": secondID})
	first.await("VOTE_ACCEPTED")
	second.send("CAST_VOTE", map[string]any{"targetPlayerId": firstID})
	second.await("VOTE_ACCEPTED")

	// 主持人直接按結束投票，不用等 60 秒走完
	host.send("END_VOTING", map[string]any{})
	result := host.await("ROUND_RESULT")
	winners, _ := result["winners"].([]any)
	if len(winners) != 2 {
		t.Fatalf("應該 2 位得獎，實際 %d", len(winners))
	}
	w1, _ := winners[0].(map[string]any)
	w2, _ := winners[1].(map[string]any)
	if w1["playerId"] != firstID || w1["rank"] != float64(1) || w1["points"] != float64(100) {
		t.Fatalf("同票時先上傳的應該第 1 名 100 分，實際 %v", w1)
	}
	if w2["playerId"] != secondID || w2["rank"] != float64(2) || w2["points"] != float64(80) {
		t.Fatalf("同票時後上傳的應該第 2 名 80 分，實際 %v", w2)
	}
}

// TestIntroAndPracticeText 開場說明與試玩題目的設定：
// 試玩題沒指定就用該模式的預設題、可以在建房或大廳改；說明翻頁會轉播給玩家；
// 一局結束回大廳後說明自動關掉，不會每一局都重播
func TestIntroAndPracticeText(t *testing.T) {
	srv := newTestServer(t)

	// ── 單人：預設「比出小愛心」，大廳改成自己打的題目
	host := srv.dial(t)
	host.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "solo", "questionMode": "random",
		"totalQuestions": 1, "questionTimeLimit": 60,
		"practiceRound": true, "gameIntro": true,
	})
	created := host.await("ROOM_CREATED")
	roomID, _ := created["roomId"].(string)
	if text, _ := created["practiceText"].(string); text != "比出小愛心" {
		t.Fatalf("單人預設試玩題應為「比出小愛心」，實際 %q", text)
	}
	if intro, _ := created["introEnabled"].(bool); !intro {
		t.Fatal("建房時 gameIntro 沒有生效")
	}

	player := srv.dial(t)
	player.send("JOIN_ROOM", map[string]any{"roomId": roomID, "playerName": "小新"})
	player.await("JOIN_SUCCESS")

	host.send("UPDATE_SETTINGS", map[string]any{"practiceText": "  踩到狗大便時的表情  "})
	updated := host.await("SETTINGS_UPDATED")
	if text, _ := updated["practiceText"].(string); text != "踩到狗大便時的表情" {
		t.Fatalf("大廳改試玩題沒生效，實際 %q", text)
	}
	if intro, _ := updated["introEnabled"].(bool); !intro {
		t.Fatal("只改試玩題不該動到說明開關")
	}
	player.await("SETTINGS_UPDATED")

	// 說明翻頁轉播給玩家
	host.send("GAME_INTRO", map[string]any{"active": true, "page": 1, "total": 5, "title": "題目出現，馬上拍！"})
	intro := player.await("GAME_INTRO")
	if page, _ := intro["page"].(float64); int(page) != 1 {
		t.Fatalf("說明頁次應為 1，實際 %v", intro["page"])
	}
	if title, _ := intro["title"].(string); title != "題目出現，馬上拍！" {
		t.Fatalf("說明標題沒轉播，實際 %q", title)
	}
	// 玩家不能假冒主持人播說明
	player.send("GAME_INTRO", map[string]any{"active": true, "page": 0, "total": 5})
	rejected := <-player.msgs
	if data, _ := rejected.Data.(map[string]any); rejected.Type != "ERROR" || data["code"] != "NOT_HOST" {
		t.Fatalf("玩家送 GAME_INTRO 應回 NOT_HOST，實際 %s %v", rejected.Type, rejected.Data)
	}

	host.send("START_GAME", map[string]any{})
	practice := host.await("NEW_QUESTION")
	if text, _ := practice["text"].(string); text != "踩到狗大便時的表情" {
		t.Fatalf("試玩題應該是主持人改的那一題，實際 %q", text)
	}

	// 遊戲開始後遲到的翻頁訊息要被丟掉，不能蓋住玩家的拍照畫面
	host.send("GAME_INTRO", map[string]any{"active": true, "page": 2, "total": 5})

	host.send("END_PRACTICE", map[string]any{})
	host.await("NEW_QUESTION")
	host.send("END_SHOOTING", map[string]any{})
	host.await("GAME_FINISHED")
	for {
		msg := <-player.msgs
		if msg.Type == "GAME_INTRO" {
			t.Fatal("遊戲開始後不該再轉播說明")
		}
		if msg.Type == "GAME_FINISHED" {
			break
		}
	}

	host.send("RESET_ROOM_TO_LOBBY", map[string]any{})
	reset := host.await("ROOM_RESET_TO_LOBBY")
	if intro, ok := reset["introEnabled"].(bool); !ok || intro {
		t.Fatalf("再來一局不該重播說明，introEnabled 實際 %v", reset["introEnabled"])
	}

	// 大廳可以再打開，也可以關掉試玩
	host.send("UPDATE_SETTINGS", map[string]any{"gameIntro": true, "practiceRound": false})
	again := host.await("SETTINGS_UPDATED")
	if intro, _ := again["introEnabled"].(bool); !intro {
		t.Fatal("大廳重新打開說明沒生效")
	}
	if practice, _ := again["practiceEnabled"].(bool); practice {
		t.Fatal("大廳關掉試玩沒生效")
	}

	// ── 團體：預設「全員一起比 YA」
	host2 := srv.dial(t)
	host2.send("CREATE_ROOM", map[string]any{
		"hostName": "主持人", "mode": "group", "questionMode": "random",
		"totalQuestions": 1, "questionTimeLimit": 60, "practiceRound": true,
	})
	created2 := host2.await("ROOM_CREATED")
	if text, _ := created2["practiceText"].(string); text != "全員一起比 YA" {
		t.Fatalf("團體預設試玩題應為「全員一起比 YA」，實際 %q", text)
	}
	if intro, _ := created2["introEnabled"].(bool); intro {
		t.Fatal("沒勾說明時 introEnabled 應為 false")
	}
}
