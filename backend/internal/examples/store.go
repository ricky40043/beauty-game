package examples

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var (
	// ErrUnsupportedType 非支援的圖片格式
	ErrUnsupportedType = errors.New("只接受 JPEG / PNG / WebP / SVG")
	// ErrTooLarge 檔案過大
	ErrTooLarge = errors.New("圖片太大")
)

// MaxOverrideBytes 後台上傳的示範圖上限
const MaxOverrideBytes = 4 << 20

// Store 房主自己補的示範圖。
//
// 這裡刻意寫到磁碟而不是記憶體：示範圖是「內容設定」而不是玩家的即時照片，
// 重開服務之後應該還在。玩家拍的照片仍然只存在記憶體、不落地。
type Store struct {
	mu  sync.RWMutex
	dir string
}

// NewStore 建立示範圖庫，dir 不存在會自動建立
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("建立示範圖資料夾失敗: %w", err)
	}
	return &Store{dir: dir}, nil
}

// Dir 圖片存放位置
func (s *Store) Dir() string { return s.dir }

func detectType(data []byte) (contentType, ext string, ok bool) {
	switch {
	case len(data) >= 3 && bytes.Equal(data[:3], []byte{0xFF, 0xD8, 0xFF}):
		return "image/jpeg", "jpg", true
	case len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return "image/png", "png", true
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return "image/webp", "webp", true
	}

	// SVG 沒有 magic bytes，看開頭有沒有 <svg
	head := strings.ToLower(string(data[:min(len(data), 512)]))
	if strings.Contains(head, "<svg") {
		return "image/svg+xml", "svg", true
	}

	return "", "", false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *Store) path(questionID int, ext string) string {
	return filepath.Join(s.dir, strconv.Itoa(questionID)+"."+ext)
}

var knownExts = []string{"jpg", "png", "webp", "svg"}

// Get 讀出房主上傳的示範圖
func (s *Store) Get(questionID int) (data []byte, contentType string, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ext := range knownExts {
		raw, err := os.ReadFile(s.path(questionID, ext))
		if err != nil {
			continue
		}
		ct, _, valid := detectType(raw)
		if !valid {
			continue
		}
		return raw, ct, true
	}

	return nil, "", false
}

// Has 是否有房主上傳的版本
func (s *Store) Has(questionID int) bool {
	_, _, ok := s.Get(questionID)
	return ok
}

// Put 存入房主上傳的示範圖，會覆蓋掉舊的
func (s *Store) Put(questionID int, data []byte) error {
	if len(data) == 0 {
		return ErrUnsupportedType
	}
	if len(data) > MaxOverrideBytes {
		return fmt.Errorf("%w（上限 %d KB）", ErrTooLarge, MaxOverrideBytes/1024)
	}

	_, ext, ok := detectType(data)
	if !ok {
		return ErrUnsupportedType
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 先清掉同一題的其他格式，避免 2001.png 與 2001.jpg 同時存在
	for _, old := range knownExts {
		if old != ext {
			_ = os.Remove(s.path(questionID, old))
		}
	}

	return os.WriteFile(s.path(questionID, ext), data, 0o644)
}

// Delete 移除房主上傳的版本，之後就會退回內建示範圖
func (s *Store) Delete(questionID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ext := range knownExts {
		_ = os.Remove(s.path(questionID, ext))
	}
}

// OverriddenIDs 目前有哪些題被房主換過圖
func (s *Store) OverriddenIDs() []int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil
	}

	ids := make([]int, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		ext := strings.TrimPrefix(filepath.Ext(name), ".")

		valid := false
		for _, known := range knownExts {
			if ext == known {
				valid = true
				break
			}
		}
		if !valid {
			continue
		}

		id, err := strconv.Atoi(strings.TrimSuffix(name, filepath.Ext(name)))
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}

	sort.Ints(ids)
	return ids
}
