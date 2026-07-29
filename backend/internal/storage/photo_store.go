package storage

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrTooLarge 單張照片超過上限
	ErrTooLarge = errors.New("照片太大")
	// ErrRoomQuota 該房間照片配額已滿
	ErrRoomQuota = errors.New("這個房間的照片已達上限")
	// ErrUnsupportedType 非支援的圖片格式
	ErrUnsupportedType = errors.New("只接受 JPEG / PNG / WebP 圖片")
	// ErrNotFound 照片不存在
	ErrNotFound = errors.New("找不到這張照片")
)

// Photo 一張存在記憶體中的照片
type Photo struct {
	ID          string
	RoomID      string
	ContentType string
	Data        []byte
	CreatedAt   time.Time
}

// Limits 儲存限制
type Limits struct {
	MaxPhotoBytes int64
	MaxRoomPhotos int
	MaxRoomBytes  int64
}

// PhotoStore 純記憶體照片庫。照片不落地，房間被清理時整批釋放。
type PhotoStore struct {
	mu     sync.RWMutex
	limits Limits

	photos    map[string]*Photo   // photoID -> 照片
	byRoom    map[string][]string // roomID  -> photoID 列表
	roomBytes map[string]int64    // roomID  -> 已用容量
}

// New 建立照片庫
func New(limits Limits) *PhotoStore {
	return &PhotoStore{
		limits:    limits,
		photos:    make(map[string]*Photo),
		byRoom:    make(map[string][]string),
		roomBytes: make(map[string]int64),
	}
}

// detectContentType 用 magic bytes 判斷格式，不信任 client 送的 Content-Type
func detectContentType(data []byte) (string, bool) {
	switch {
	case len(data) >= 3 && bytes.Equal(data[:3], []byte{0xFF, 0xD8, 0xFF}):
		return "image/jpeg", true
	case len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return "image/png", true
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return "image/webp", true
	default:
		return "", false
	}
}

// Put 存入一張照片，回傳 photoID
func (s *PhotoStore) Put(roomID string, data []byte) (*Photo, error) {
	size := int64(len(data))
	if size == 0 {
		return nil, ErrUnsupportedType
	}
	if size > s.limits.MaxPhotoBytes {
		return nil, fmt.Errorf("%w（上限 %d KB）", ErrTooLarge, s.limits.MaxPhotoBytes/1024)
	}

	contentType, ok := detectContentType(data)
	if !ok {
		return nil, ErrUnsupportedType
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.byRoom[roomID]) >= s.limits.MaxRoomPhotos || s.roomBytes[roomID]+size > s.limits.MaxRoomBytes {
		return nil, ErrRoomQuota
	}

	photo := &Photo{
		ID:          uuid.New().String(),
		RoomID:      roomID,
		ContentType: contentType,
		Data:        data,
		CreatedAt:   time.Now(),
	}

	s.photos[photo.ID] = photo
	s.byRoom[roomID] = append(s.byRoom[roomID], photo.ID)
	s.roomBytes[roomID] += size

	return photo, nil
}

// Get 取出照片
func (s *PhotoStore) Get(photoID string) (*Photo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	photo, ok := s.photos[photoID]
	if !ok {
		return nil, ErrNotFound
	}
	return photo, nil
}

// Delete 刪掉單張照片（例如玩家重新上傳，舊照片就沒用了）
func (s *PhotoStore) Delete(photoID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	photo, ok := s.photos[photoID]
	if !ok {
		return
	}

	delete(s.photos, photoID)
	s.roomBytes[photo.RoomID] -= int64(len(photo.Data))
	if s.roomBytes[photo.RoomID] < 0 {
		s.roomBytes[photo.RoomID] = 0
	}

	ids := s.byRoom[photo.RoomID]
	for i, id := range ids {
		if id == photoID {
			s.byRoom[photo.RoomID] = append(ids[:i], ids[i+1:]...)
			break
		}
	}
}

// PurgeRoom 房間結束或被清理時，釋放它的所有照片
func (s *PhotoStore) PurgeRoom(roomID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := s.byRoom[roomID]
	for _, id := range ids {
		delete(s.photos, id)
	}
	delete(s.byRoom, roomID)
	delete(s.roomBytes, roomID)

	return len(ids)
}

// Stats 目前用量，給 /api/health 觀察用
func (s *PhotoStore) Stats() (photos int, rooms int, bytes int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, b := range s.roomBytes {
		bytes += b
	}
	return len(s.photos), len(s.byRoom), bytes
}
