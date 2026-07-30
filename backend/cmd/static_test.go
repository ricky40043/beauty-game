package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestStaticFileServesRealFiles ./static 底下存在的檔案要找得到。
// 之前是逐一註冊路由，漏掉的路徑（例如 /sounds/shutter.mp3）會被 SPA fallback
// 吃掉、回傳 index.html，前端拿到 HTML 當 MP3 解碼，失敗得毫無線索。
func TestStaticFileServesRealFiles(t *testing.T) {
	root := t.TempDir()
	staticRoot = root
	t.Cleanup(func() { staticRoot = "./static" })

	if err := os.MkdirAll(filepath.Join(root, "sounds"), 0o755); err != nil {
		t.Fatalf("建立測試目錄失敗: %v", err)
	}
	for _, name := range []string{"index.html", "favicon.svg", "sounds/shutter.mp3"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("建立測試檔失敗: %v", err)
		}
	}

	for _, urlPath := range []string{"/favicon.svg", "/sounds/shutter.mp3", "/index.html"} {
		if _, ok := staticFile(urlPath); !ok {
			t.Errorf("%s 應該要被當成靜態檔送出", urlPath)
		}
	}

	// SPA 路由與目錄本身都不是檔案，要交給 index.html 處理
	for _, urlPath := range []string{"/", "/lobby/ABC123", "/game/host/ABC123", "/sounds"} {
		if _, ok := staticFile(urlPath); ok {
			t.Errorf("%s 不該被當成靜態檔", urlPath)
		}
	}
}

// TestStaticFileBlocksTraversal 不能靠上層 mux 幫忙擋，這裡自己也要擋住 ..
func TestStaticFileBlocksTraversal(t *testing.T) {
	root := t.TempDir()
	staticRoot = filepath.Join(root, "static")
	t.Cleanup(func() { staticRoot = "./static" })

	if err := os.MkdirAll(staticRoot, 0o755); err != nil {
		t.Fatalf("建立測試目錄失敗: %v", err)
	}
	// 放一個「不該被讀到」的檔案在 static 外面，模擬執行檔或設定檔
	secret := filepath.Join(root, "main")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatalf("建立測試檔失敗: %v", err)
	}

	for _, urlPath := range []string{
		"/../main",
		"/../../main",
		"/sounds/../../main",
		"/./../main",
		"../main",
	} {
		got, ok := staticFile(urlPath)
		if ok {
			t.Errorf("%s 不該解析成檔案，卻得到 %s", urlPath, got)
		}
	}
}
