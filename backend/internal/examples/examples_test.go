package examples_test

import (
	"strings"
	"testing"

	"beauty-game/internal/examples"
	"beauty-game/internal/services"
)

// TestEveryGroupQuestionHasExample 每一題團體題都要有內建示範圖，
// 不然主畫面會空一塊。新增團體題時這個測試會提醒你補圖。
func TestEveryGroupQuestionHasExample(t *testing.T) {
	bank := services.AllQuestions("group")
	if len(bank) == 0 {
		t.Fatal("團體題庫是空的")
	}

	for _, q := range bank {
		if !examples.HasBuiltin(q.ID) {
			t.Errorf("團體題 %d「%s」沒有內建示範圖", q.ID, q.Text)
		}
	}
}

// TestSoloQuestionsHaveNoExample 單人題目前刻意不做圖，避免誤以為漏了
func TestSoloQuestionsHaveNoExample(t *testing.T) {
	for _, q := range services.AllQuestions("solo") {
		if examples.HasBuiltin(q.ID) {
			t.Errorf("單人題 %d 不該有內建示範圖", q.ID)
		}
	}
}

// TestRenderedSVGIsWellFormed 產出的 SVG 要是合法可用的內容
func TestRenderedSVGIsWellFormed(t *testing.T) {
	for _, id := range examples.BuiltinIDs() {
		svg, ok := examples.Builtin(id)
		if !ok {
			t.Fatalf("題目 %d 應該有示範圖", id)
		}

		switch {
		case !strings.HasPrefix(svg, "<svg"):
			t.Errorf("題目 %d 的 SVG 開頭不正確", id)
		case !strings.HasSuffix(svg, "</svg>"):
			t.Errorf("題目 %d 的 SVG 沒有正確結尾", id)
		case strings.Count(svg, "<g") != strings.Count(svg, "</g>"):
			t.Errorf("題目 %d 的 <g> 標籤沒有配對", id)
		case !strings.Contains(svg, "<title>"):
			t.Errorf("題目 %d 缺少 title，螢幕閱讀器讀不到", id)
		case strings.Contains(svg, "<circle") == false:
			t.Errorf("題目 %d 沒有畫出任何人（缺少頭部）", id)
		}
	}
}

// TestStoreOverridesBuiltin 房主上傳的圖要蓋掉內建圖，刪掉之後還原
func TestStoreOverridesBuiltin(t *testing.T) {
	store, err := examples.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("建立 store 失敗: %v", err)
	}

	const id = 2001
	if store.Has(id) {
		t.Fatal("全新的 store 不該有任何覆蓋圖")
	}

	png := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0}
	if err := store.Put(id, png); err != nil {
		t.Fatalf("存入失敗: %v", err)
	}

	data, contentType, ok := store.Get(id)
	if !ok {
		t.Fatal("存進去卻讀不出來")
	}
	if contentType != "image/png" {
		t.Fatalf("格式判斷錯誤: %s", contentType)
	}
	if len(data) != len(png) {
		t.Fatal("讀出來的內容跟存進去的不一樣")
	}

	// 換成 JPEG，舊的 PNG 不該留下來變成兩個檔案
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0, 0, 0}
	if err := store.Put(id, jpeg); err != nil {
		t.Fatalf("覆蓋失敗: %v", err)
	}
	if _, ct, _ := store.Get(id); ct != "image/jpeg" {
		t.Fatalf("覆蓋後格式應為 jpeg，實際 %s", ct)
	}
	if ids := store.OverriddenIDs(); len(ids) != 1 {
		t.Fatalf("同一題應該只留一個檔案，實際 %d 個", len(ids))
	}

	store.Delete(id)
	if store.Has(id) {
		t.Fatal("刪除後不該還讀得到")
	}
	// 刪掉自訂圖之後，內建圖要能接手
	if !examples.HasBuiltin(id) {
		t.Fatal("刪除自訂圖後應該還有內建圖可用")
	}
}

// TestStoreRejectsNonImage 非圖片內容要被擋下來
func TestStoreRejectsNonImage(t *testing.T) {
	store, err := examples.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("建立 store 失敗: %v", err)
	}

	if err := store.Put(2002, []byte("<html>not an image</html>")); err == nil {
		t.Fatal("HTML 不該被當成圖片接受")
	}
	if err := store.Put(2002, nil); err == nil {
		t.Fatal("空內容不該被接受")
	}
}

// TestStoreAcceptsSVG SVG 沒有 magic bytes，要靠內容判斷
func TestStoreAcceptsSVG(t *testing.T) {
	store, err := examples.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("建立 store 失敗: %v", err)
	}

	svg := []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	if err := store.Put(2003, svg); err != nil {
		t.Fatalf("SVG 應該可以接受: %v", err)
	}

	if _, ct, _ := store.Get(2003); ct != "image/svg+xml" {
		t.Fatalf("格式應為 image/svg+xml，實際 %s", ct)
	}
}
