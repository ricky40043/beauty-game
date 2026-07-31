package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"beauty-game/internal/config"
	"beauty-game/internal/examples"
	"beauty-game/internal/handlers"
	"beauty-game/internal/services"
	"beauty-game/internal/storage"
	"beauty-game/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// staticRoot 前端 build 產物的位置。宣告成變數是為了讓測試能指到暫存目錄。
var staticRoot = "./static"

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("提示: 沒有 .env 檔，改用系統環境變數")
	}

	cfg := config.Load()
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 這個遊戲完全跑在記憶體上：房間、分數、照片都不落地
	photoStore := storage.New(storage.Limits{
		MaxPhotoBytes: cfg.Photo.MaxPhotoBytes,
		MaxRoomPhotos: cfg.Photo.MaxRoomPhotos,
		MaxRoomBytes:  cfg.Photo.MaxRoomBytes,
	})

	// 示範圖：內建的火柴人 SVG 存在程式裡，房主補的圖寫到磁碟，重啟後還在
	exampleStore, err := examples.NewStore(cfg.ExampleDir)
	if err != nil {
		log.Fatalf("❌ 無法初始化示範圖資料夾: %v", err)
	}
	exampleStore.SeedFrom("./data/examples")

	// 自訂題目跟示範圖放同一個資料夾，一起被 volume 保護，重新部署不會消失
	questionStore, err := services.NewQuestionStore(cfg.ExampleDir)
	if err != nil {
		log.Fatalf("❌ 無法初始化題目設定: %v", err)
	}

	questionService := services.NewQuestionService(questionStore)

	// 把資料實際落在哪、載到幾筆印出來。
	//
	// 自訂題目與示範圖是這個服務唯一會寫進磁碟的東西，其他（房間、分數、玩家
	// 照片）都只在記憶體，重啟就沒了 —— 這是刻意的。但也因此，只要 EXAMPLE_DIR
	// 指到 volume 外面，每次部署就會靜靜地把內容清光而沒有任何錯誤訊息。
	// 印出絕對路徑與筆數，之後懷疑資料掉了，看這一行就能判斷。
	absExampleDir, err := filepath.Abs(cfg.ExampleDir)
	if err != nil {
		absExampleDir = cfg.ExampleDir
	}
	log.Printf("💾 資料目錄 %s（自訂題目 %d 題、自訂示範圖 %d 張）",
		absExampleDir, questionStore.Count(), len(exampleStore.OverriddenIDs()))
	if !filepath.IsAbs(cfg.ExampleDir) {
		log.Printf("⚠️ EXAMPLE_DIR 是相對路徑，會跟著工作目錄跑。容器部署請設成掛載點底下的絕對路徑，否則重新部署資料會不見")
	}
	gameService := services.NewGameService()
	roomService := services.NewRoomService(cfg, questionService)

	hub := websocket.NewHub(roomService, gameService, questionService, photoStore, cfg.FrontendURL)
	go hub.Run()

	apiHandler := handlers.NewAPIHandler(roomService, questionService, hub)
	photoHandler := handlers.NewPhotoHandler(photoStore, roomService, cfg.Photo.MaxPhotoBytes)
	exampleHandler := handlers.NewExampleHandler(exampleStore, questionService, cfg.AdminToken)

	router := setupRoutes(cfg, apiHandler, photoHandler, exampleHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	go func() {
		log.Printf("💄 今日我最美 啟動於 http://%s:%s", cfg.Host, cfg.Port)
		log.Printf("📡 WebSocket: ws://%s:%s/ws", cfg.Host, cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ 服務啟動失敗: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🔄 正在關閉服務…")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("❌ 關閉失敗: %v", err)
	}
	log.Println("✅ 已關閉")
}

func setupRoutes(
	cfg *config.Config,
	api *handlers.APIHandler,
	photo *handlers.PhotoHandler,
	example *handlers.ExampleHandler,
) *gin.Engine {
	router := gin.Default()
	router.Use(corsMiddleware(cfg))
	router.Use(cacheMiddleware())

	router.GET("/api/health", api.Health)

	group := router.Group("/api")
	{
		group.GET("/questions", api.GetQuestions)
		group.GET("/rooms/:roomId", api.GetRoom)
		group.POST("/rooms/:roomId/photos", photo.Upload)
		group.GET("/photos/:photoId", photo.Get)

		// 題目示範圖：讀取公開，後台寫入需要 ADMIN_TOKEN（沒設就不驗）
		group.GET("/questions/:questionId/example", example.Get)

		admin := group.Group("/admin", example.RequireAdmin)
		{
			admin.GET("/examples", example.List)
			admin.POST("/questions/:questionId/example", example.Upload)
			admin.DELETE("/questions/:questionId/example", example.Delete)

			// 題目管理：新增自己的題目、修改、刪除，內建題可以停用
			admin.POST("/questions", example.CreateQuestion)
			admin.PUT("/questions/:questionId", example.UpdateQuestion)
			admin.DELETE("/questions/:questionId", example.DeleteQuestion)
			admin.POST("/questions/:questionId/disabled", example.SetQuestionDisabled)
		}
	}

	router.GET("/ws", api.ServeWS)

	// 前後端合一部署：Vue build 出來的檔案放在 ./static
	router.Static("/assets", "./static/assets")

	router.NoRoute(func(c *gin.Context) {
		// ./static 底下真的存在這個檔案就直接送出（favicon、/sounds/*.mp3、
		// 之後放進 public/ 的任何東西都適用），否則才交給 SPA 的 index.html。
		//
		// 原本是逐一註冊每個靜態檔，漏掉的路徑會靜靜地回傳 index.html —— 前端拿到
		// 一份 HTML 當成 MP3 去解碼，失敗得毫無線索。改成看檔案存不存在就不會再犯。
		if file, ok := staticFile(c.Request.URL.Path); ok {
			c.File(file)
			return
		}
		c.File("./static/index.html")
	})

	return router
}

// cacheMiddleware 決定靜態內容的快取策略。
//
// 沒有這段的話 index.html 不帶 Cache-Control，瀏覽器會用「啟發式快取」自己決定
// 存多久，使用者於是一直拿到舊版的 index —— 而它指向的是上一版的 chunk 檔名，
// 等於每次部署都有人看到舊畫面。
// staticFile 把網址路徑對應到 ./static 底下的實體檔案。
// path.Clean 會把 ".." 收斂回根目錄，藉此擋掉目錄穿越。
func staticFile(urlPath string) (string, bool) {
	clean := path.Clean("/" + strings.TrimPrefix(urlPath, "/"))
	if clean == "/" {
		return "", false
	}

	full := filepath.Join(staticRoot, filepath.FromSlash(clean))
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		return "", false
	}
	return full, true
}

func cacheMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		switch {
		case strings.HasPrefix(path, "/api/"):
			// 交給各 handler 自己決定（照片、示範圖有各自的策略）

		case strings.HasPrefix(path, "/assets/"):
			// Vite 的檔名帶內容雜湊，內容一改檔名就變，可以放心永久快取
			c.Header("Cache-Control", "public, max-age=31536000, immutable")

		default:
			// index.html 與所有 SPA 路由：每次都要回來確認有沒有新版。
			// 檔案沒變時仍會走 304，成本很低。
			c.Header("Cache-Control", "no-cache")
		}

		c.Next()
	}
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if cfg.Environment != "production" {
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			for _, allowed := range cfg.CORSOrigins {
				if origin == allowed {
					c.Header("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
