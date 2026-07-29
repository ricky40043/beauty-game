package config

import (
	"os"
	"strconv"
	"strings"
)

// Config 應用程式配置，全部來自環境變數並帶預設值
type Config struct {
	Port        string
	Host        string
	Environment string
	FrontendURL string
	CORSOrigins []string

	Game  GameConfig
	Photo PhotoConfig

	// ExampleDir 題目示範圖的存放位置；AdminToken 為空代表後台不設密碼
	ExampleDir string
	AdminToken string
}

// GameConfig 遊戲相關預設值
type GameConfig struct {
	MaxPlayersPerRoom     int
	RoomIDLength          int
	QuestionTimeLimit     int // 每題拍照秒數
	DefaultTotalQuestions int
}

// PhotoConfig 照片儲存限制
type PhotoConfig struct {
	MaxPhotoBytes int64 // 單張照片上限
	MaxRoomPhotos int   // 每間房照片張數上限
	MaxRoomBytes  int64 // 每間房照片總容量上限
}

// Load 載入配置
func Load() *Config {
	frontendURL := strings.TrimSuffix(getEnv("FRONTEND_URL", "http://localhost:3344"), "/")
	corsOrigins := getEnvAsSlice("CORS_ORIGINS", []string{
		"http://localhost:3344",
		"http://localhost:5173",
	})

	if frontendURL != "" {
		matched := false
		for _, origin := range corsOrigins {
			if strings.TrimSuffix(origin, "/") == frontendURL {
				matched = true
				break
			}
		}
		if !matched {
			corsOrigins = append(corsOrigins, frontendURL)
		}
	}

	return &Config{
		Port:        getEnv("PORT", "8081"),
		Host:        getEnv("HOST", "localhost"),
		Environment: getEnv("ENV", "development"),
		FrontendURL: frontendURL,
		CORSOrigins: corsOrigins,

		Game: GameConfig{
			MaxPlayersPerRoom:     getEnvAsInt("MAX_PLAYERS_PER_ROOM", 30),
			RoomIDLength:          getEnvAsInt("ROOM_ID_LENGTH", 6),
			QuestionTimeLimit:     getEnvAsInt("QUESTION_TIME_LIMIT", 60),
			DefaultTotalQuestions: getEnvAsInt("DEFAULT_TOTAL_QUESTIONS", 10),
		},

		ExampleDir: getEnv("EXAMPLE_DIR", "./data/examples"),
		AdminToken: getEnv("ADMIN_TOKEN", ""),

		Photo: PhotoConfig{
			MaxPhotoBytes: int64(getEnvAsInt("MAX_PHOTO_BYTES", 8<<20)),
			MaxRoomPhotos: getEnvAsInt("MAX_ROOM_PHOTOS", 300),
			MaxRoomBytes:  int64(getEnvAsInt("MAX_ROOM_BYTES", 80<<20)),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if valueStr := os.Getenv(key); valueStr != "" {
		return strings.Split(valueStr, ",")
	}
	return defaultValue
}
