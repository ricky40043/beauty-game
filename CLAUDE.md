# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

「今日我最美」— a photo-based party game. The host opens a big screen (TV/laptop), players join by scanning a QR code on their phones. Each round shows a mission ("拍出你最兇的臉"), players shoot and upload a photo, submissions pop onto the host screen in real time, and the host picks 1–5 winners before advancing. Go WebSocket backend + Vue 3 frontend, all in Traditional Chinese (zh-TW).

Everything runs in memory — no database, no Redis, no disk. Photos live in a per-room in-memory store and are purged when the room is reset, closed, or idle-cleaned.

## Development Commands

```bash
./start.sh                       # 後端 :8081 + 前端 :3344，一起起

cd backend && go run cmd/main.go # 只跑後端
cd backend && go test ./...      # 整合測試（約 15 秒，含一題 15 秒倒數的真實等待）
cd backend && gofmt -w ./cmd ./internal

cd frontend && npm run dev       # Vite dev server，proxy /api 與 /ws 到 :8081
cd frontend && npm run build     # vue-tsc 型別檢查 + production build
cd frontend && npm run type-check
```

`backend/static/` 是前端 build 的產物複本（已被 gitignore）。存在時後端單獨跑就能在 `:8081` 提供完整 app；改了前端要重新 `npm run build && cp -r frontend/dist backend/static` 才會更新。

## Architecture

### Backend (`backend/`)

**Hub-and-Spoke WebSocket：**
- `internal/websocket/hub.go` — Hub 管理連線、房間廣播群組、閒置房間清理（2 小時）
- `internal/websocket/client.go` — 每條連線一個 Client（`readPump`/`writePump`），房間類訊息處理
- `internal/websocket/round.go` — 回合流程：出題、每秒倒數、收桌、評選、下一題、結算

**服務層：**
- `internal/services/game_service.go` — 回合狀態機與計分。**所有會改動 Room 的動作都必須經過這裡的 mutex**，Hub 與 Client 不直接改 Room 欄位
- `internal/services/room_service.go` — 記憶體房間 CRUD、玩家進出、自動暱稱、房號產生（去掉 0/O/1/I）
- `internal/services/question_service.go` / `question_bank.go` — 內建 75 題題庫（solo 50 / group 25）、隨機抽題、自選題序組題
- `internal/services/question_store.go` — 後台自訂題目與停用清單，落地成 `{EXAMPLE_DIR}/questions.json`（先寫暫存檔再 rename）
- `internal/storage/photo_store.go` — 記憶體照片庫，magic bytes 驗格式，per-room 張數與容量配額
- `internal/examples/` — 題目示範圖。`figure.go` 是火柴人 SVG 產生器（肢體用 polyline 畫出關節，頭與臉最後畫才不會被手臂穿過），`scenes.go` 是 25 題團體題的站位設定，`store.go` 是房主自訂圖的磁碟覆寫層
- `internal/handlers/example_handler.go` — 示範圖讀取（自訂圖優先、退回內建、都沒有 404）與後台上傳/刪除
- `internal/names/names.go` — 免取名模式的「形容詞 + 動物 + emoji」暱稱產生器

**入口：** `cmd/main.go` — config → photoStore → services → Hub → Gin routes → graceful shutdown

### Frontend (`frontend/src/`)

**Stores（Pinia）：**
- `stores/socket.ts` — WebSocket 連線、指數退避重連、localStorage session（`beauty_game_session`）恢復、訊息路由與跳頁
- `stores/game.ts` — 房間、我是誰、當前題目、本回合照片、分數、歷史
- `stores/ui.ts` — toast

**一次性事件走 `utils/bus.ts`**（照片彈出、得獎公布），不塞進 store 狀態。

**Views：** Home / CreateRoom / JoinRoom / Lobby / GameHost / GamePlayer / Results / NotFound
主畫面（房主）與手機端（玩家）共用 `LobbyView`，用 `isHost` 分岔成兩套版面。

**關鍵元件：**
- `CameraCapture.vue` — 主路徑 `getUserMedia` 全螢幕預覽 + 前後鏡頭切換；不可用時自動退回 `<input type="file" capture>`。拍完進預覽可重拍
- `PhotoPopupLayer.vue` — 照片彈出停 **3 秒**淡出，隨機 **±30 度**傾斜疊成相片堆。傾斜角走 CSS 變數 `--tilt`，否則進場動畫的 `scale()` 會把旋轉整個覆蓋掉
- `PhotoStrip.vue` — 主畫面常駐「最快前 5 張」縮圖，可點開燈箱
- `JudgePanel.vue` — 房主評選，點選順序即名次
- `QuestionPicker.vue` — 建房與大廳共用的題目選擇器（分類、搜尋、排序、自訂題）

## 題庫來源

實際出題的題庫 = **內建題（編譯進程式）+ 後台自訂題（存磁碟）− 被停用的**，由
`QuestionService.GetBank()` 合併。專案**沒有資料庫**，自訂題目就是一個 JSON 檔。

編號分段刻意錯開，看編號就知道來源：

| 範圍 | 用途 |
|---|---|
| 1001~1508 | 內建單人題 |
| 2001~2025 | 內建團體題 |
| 9000+ | 房主建房時臨時打的題目（不落地，只存在該房間） |
| 9999 | 試玩題 |
| 30000+ | 後台新增的自訂題（落地） |

`questions.json` 存了 `nextId` 並且只增不減。**不能改用「現有最大值 +1」** ——
示範圖是以題號當檔名，編號被回收的話新題目會接到上一題留下來的圖。

內建題不能改也不能刪（它們在執行檔裡），但可以停用讓它不再被抽到。
`services.AllQuestions()` 一律只回內建題，後台列表要靠它才看得到被停用的項目
（`GetBank()` 已經濾掉了）。

## 題目示範圖

`GET /api/questions/:id/example` 依序找：房主上傳的自訂圖 → 內建 SVG → 404。
前端不預先問「這題有沒有圖」，直接載入、`@error` 時把區塊收起來，房主之後補圖會自動生效。

只有團體題有內建圖（`internal/examples/scenes.go`），`examples_test.go` 會確保**每一題團體題都有圖**，新增團體題忘記補圖時測試會失敗。

後台在 `/admin`，寫入操作由 `ADMIN_TOKEN` 保護（未設則不驗證）。自訂圖寫到 `EXAMPLE_DIR` 磁碟 —— 這是刻意的：示範圖屬於「內容設定」，重啟後應該還在；玩家拍的照片仍然只在記憶體、不落地。

**不從網路抓圖**：版權素材不可打包進會對外部署的 App，內建圖一律程式自己畫。

## 試玩回合

建房可勾選「開場先試玩一輪」（`Room.PracticeEnabled`）。開始遊戲後先進 `InPractice` 狀態：
用固定的 `models.PracticeQuestion`、**不倒數**（`TimeLeft = 0`、不啟計時器）、**不計分**、
**不自動收桌**（`AllPlayersSubmitted` 在試玩時一律回 false），主畫面顯示全部投稿而非前 5 張。
房主按 `END_PRACTICE` 才清掉試玩照片並進正式第 1 題。

## 計分規則（`services/game_service.go`）

- 每人每題只有一張投稿，再傳就是覆蓋（沿用抵達順序、不重複給分）
- 完成上傳 +10
- 抵達順序前三 +15 / +10 / +5
- 得獎依房主點選順序 100 / 80 / 60 / 40 / 20
- 團體模式該回合有得獎時，全場每人再 +50 合作分

## WebSocket 協定

**Client → Server：** `CREATE_ROOM`、`JOIN_ROOM`、`REJOIN_ROOM`、`CHECK_ROOM`、`RENAME_PLAYER`、`UPDATE_SETTINGS`、`START_GAME`、`SUBMIT_PHOTO`、`END_SHOOTING`、`PICK_WINNERS`、`SKIP_ROUND`、`NEXT_QUESTION`、`RESET_ROOM_TO_LOBBY`、`LEAVE_ROOM`、`PING`

**Server → Client：** `ROOM_CREATED`、`JOIN_SUCCESS`、`PLAYER_JOINED/LEFT/DISCONNECTED/REJOINED/RENAMED`、`ROOM_STATUS`、`REJOIN_SUCCESS`、`SETTINGS_UPDATED`、`GAME_STARTED`、`NEW_QUESTION`、`TIMER_UPDATE`、`PHOTO_SUBMITTED`、`PHOTO_ACCEPTED`、`ROUND_CLOSED`、`ROUND_RESULT`、`GAME_FINISHED`、`ROOM_RESET_TO_LOBBY`、`ROOM_CLOSED`、`HOST_DISCONNECTED/RECONNECTED`、`ERROR`、`PONG`

**REST 端點：** `/api/health`、`/api/questions`、`/api/questions/:id/example`、`/api/rooms/:roomId`、`/api/rooms/:roomId/photos`、`/api/photos/:photoId`、`/api/admin/examples`、`/api/admin/questions/:id/example`

**照片不走 WebSocket。** 前端壓縮後 `POST /api/rooms/:roomId/photos`（multipart，欄位 `photo`）拿到 `photoId`，再用 `SUBMIT_PHOTO` 送 id；`maxMessageSize` 因此只有 8KB。

## 記憶體與磁碟的成長界線

會寫進磁碟的只有 `{EXAMPLE_DIR}` 底下的自訂題目（`questions.json`）與自訂示範圖，
兩者都被題數綁住：題目每筆約 150 bytes、示範圖每題最多一張且上限 4MB。不會無限成長。

玩家照片只在記憶體，每間房有 `MAX_ROOM_PHOTOS` 與 `MAX_ROOM_BYTES` 上限
（程式預設 300 張 / 80MB，地端 compose 放大到 400 張 / 512MB）。
**換題時必須呼叫 `releaseRoundPhotos`**：它只清 `room.RoundPhotos` 這個列表是不夠的，
照片實體留在照片庫裡會累積整場，20 人玩到第 15 題左右就把配額吃光，玩家上傳開始收到
507「這個房間的照片已達上限」。得獎照片要保留 —— 結算頁的照片牆靠它們。

抓配額時**張數比容量更容易先撞到**：一間房的用量是「累積的得獎照片 + 當前這題的照片」，
50 題 × 每題 5 名 = 250 張得獎，加上當前題 30 人就是 280 張，兩個上限要一起調。

配額是在 **request body 整包讀進記憶體之後**才檢查的（`ParseMultipartForm` 一份、
`io.ReadAll` 再一份），所以被 507 擋掉的上傳一樣付了完整的頻寬與記憶體。
實測 16 條併發 8MB 上傳、只存下 80MB，RSS 就衝到 1.1GB —— 部署時要留這個瞬間放大的餘裕，
`docker-compose.yml` 的 `mem_limit` 是為此設的。

## 重要限制

- 手機端用 `h-[100dvh]` + `overflow-hidden` 鎖成一頁。`min-h-screen` 會讓內容超出視窗、底下留一塊黑的，而且網址列伸縮時高度不會跟著變
- **`getUserMedia` 需要 HTTPS 或 localhost。** 手機用區網 IP 走 http 連 dev server 時網頁相機會被擋，`CameraCapture` 會自動退回系統相機，功能不受影響
- 房主不是玩家：房主只負責主畫面與評選，不參與拍照與計分
- 房主身分靠 `hostToken`（localStorage）驗證，換連線重連後仍是房主

## Coding Conventions

- **Vue：** `<script setup lang="ts">` + Composition API
- **Go：** gofmt、錯誤一律處理、註解寫「為什麼」而非「做什麼」
- **CSS：** Tailwind utilities；共用樣式放 `style.css` 的 `@layer components`（`.btn` / `.card` / `.field`）
- **命名：** Vue 元件 PascalCase、TS 檔 camelCase、Go 檔 snake_case
- 所有面向使用者的文字都是繁體中文

## Deployment

單一 Docker image（前端 build → 後端 build + **測試當關卡** → alpine），`render.yaml` 可直接 Render Blueprint 部署。`FRONTEND_URL` 決定 QR code 指向哪裡。
