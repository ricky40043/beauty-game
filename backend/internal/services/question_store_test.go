package services_test

import (
	"testing"

	"beauty-game/internal/models"
	"beauty-game/internal/services"
)

func newStore(t *testing.T) *services.QuestionStore {
	t.Helper()
	store, err := services.NewQuestionStore(t.TempDir())
	if err != nil {
		t.Fatalf("建立題目儲存失敗: %v", err)
	}
	return store
}

// TestAddQuestionPersists 新增的題目要寫進檔案，重開之後還在
func TestAddQuestionPersists(t *testing.T) {
	dir := t.TempDir()

	store, err := services.NewQuestionStore(dir)
	if err != nil {
		t.Fatalf("建立失敗: %v", err)
	}

	created, err := store.Add("拍一張你最得意的表情", models.ModeSolo, models.CategoryExpression, 2)
	if err != nil {
		t.Fatalf("新增失敗: %v", err)
	}
	if created.ID < services.CustomQuestionIDBase {
		t.Fatalf("自訂題目編號應該從 %d 起算，實際 %d", services.CustomQuestionIDBase, created.ID)
	}
	if !created.IsCustom {
		t.Fatal("自訂題目應該標記 IsCustom")
	}

	// 換一個 store 實例讀同一個資料夾，模擬重新啟動服務
	reopened, err := services.NewQuestionStore(dir)
	if err != nil {
		t.Fatalf("重新開啟失敗: %v", err)
	}

	got := reopened.Custom(models.ModeSolo)
	if len(got) != 1 || got[0].Text != "拍一張你最得意的表情" {
		t.Fatalf("重開後應該讀得到剛才那一題，實際 %+v", got)
	}
	if got[0].Difficulty != 2 {
		t.Fatalf("難度應該保留為 2，實際 %d", got[0].Difficulty)
	}
}

// TestAddQuestionValidation 空白題目要被擋，過長要截斷
func TestAddQuestionValidation(t *testing.T) {
	store := newStore(t)

	if _, err := store.Add("   ", models.ModeSolo, "", 1); err == nil {
		t.Fatal("空白題目不該被接受")
	}

	long := ""
	for i := 0; i < 200; i++ {
		long += "題"
	}
	created, err := store.Add(long, models.ModeGroup, "", 1)
	if err != nil {
		t.Fatalf("新增失敗: %v", err)
	}
	if runes := []rune(created.Text); len(runes) != 80 {
		t.Fatalf("超長題目應被截到 80 字，實際 %d", len(runes))
	}
	// 團體模式沒指定分類時要落到 group，前端才會顯示正確標籤
	if created.Category != models.CategoryGroup {
		t.Fatalf("團體題預設分類應為 group，實際 %s", created.Category)
	}
}

// TestUpdateAndDeleteCustom 自訂題目可以改也可以刪
func TestUpdateAndDeleteCustom(t *testing.T) {
	store := newStore(t)

	created, _ := store.Add("原本的題目", models.ModeSolo, models.CategoryPose, 1)

	updated, err := store.Update(created.ID, "改過的題目", models.CategoryObject, 2)
	if err != nil {
		t.Fatalf("修改失敗: %v", err)
	}
	if updated.Text != "改過的題目" || updated.Category != models.CategoryObject || updated.Difficulty != 2 {
		t.Fatalf("修改結果不對: %+v", updated)
	}

	if err := store.Delete(created.ID); err != nil {
		t.Fatalf("刪除失敗: %v", err)
	}
	if len(store.Custom(models.ModeSolo)) != 0 {
		t.Fatal("刪除後不該還在")
	}
	if err := store.Delete(created.ID); err == nil {
		t.Fatal("刪除不存在的題目應該報錯")
	}
}

// TestBuiltinCannotBeEditedOrDeleted 內建題是編譯進去的，只能停用
func TestBuiltinCannotBeEditedOrDeleted(t *testing.T) {
	store := newStore(t)

	const builtinID = 1001
	if _, err := store.Update(builtinID, "想改內建題", "", 1); err == nil {
		t.Fatal("內建題不該可以修改")
	}
	if err := store.Delete(builtinID); err == nil {
		t.Fatal("內建題不該可以刪除")
	}

	if err := store.SetDisabled(builtinID, true); err != nil {
		t.Fatalf("停用失敗: %v", err)
	}
	if !store.IsDisabled(builtinID) {
		t.Fatal("停用後 IsDisabled 應為 true")
	}
}

// TestIDsDoNotCollideAfterDelete 刪掉再新增不能重用舊編號，
// 否則新題目會接到上一題留下來的示範圖（示範圖是用題號當檔名）
func TestIDsDoNotCollideAfterDelete(t *testing.T) {
	store := newStore(t)

	first, _ := store.Add("第一題", models.ModeSolo, "", 1)
	second, _ := store.Add("第二題", models.ModeSolo, "", 1)

	if err := store.Delete(second.ID); err != nil {
		t.Fatalf("刪除失敗: %v", err)
	}

	third, _ := store.Add("第三題", models.ModeSolo, "", 1)
	if third.ID == second.ID {
		t.Fatalf("刪除後新增不該重用編號 %d", third.ID)
	}
	if third.ID <= first.ID {
		t.Fatalf("編號應該持續遞增，first=%d third=%d", first.ID, third.ID)
	}
}

// TestServiceMergesCustomAndSkipsDisabled 出題時要看得到自訂題、看不到停用題
func TestServiceMergesCustomAndSkipsDisabled(t *testing.T) {
	store := newStore(t)
	svc := services.NewQuestionService(store)

	builtinCount := len(services.AllQuestions(models.ModeGroup))

	added, _ := store.Add("全體比出勝利手勢", models.ModeGroup, "", 1)

	bank := svc.GetBank(models.ModeGroup)
	if len(bank) != builtinCount+1 {
		t.Fatalf("題庫應為內建 %d + 自訂 1 題，實際 %d", builtinCount, len(bank))
	}

	found := false
	for _, q := range bank {
		if q.ID == added.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("自訂題目沒有出現在題庫裡")
	}

	// 停用內建的第一題與剛才那題自訂題
	_ = store.SetDisabled(services.AllQuestions(models.ModeGroup)[0].ID, true)
	_ = store.SetDisabled(added.ID, true)

	bank = svc.GetBank(models.ModeGroup)
	if len(bank) != builtinCount-1 {
		t.Fatalf("停用兩題後應剩 %d 題，實際 %d", builtinCount-1, len(bank))
	}
	for _, q := range bank {
		if q.ID == added.ID {
			t.Fatal("被停用的題目不該出現在題庫裡")
		}
	}
}

// TestBuildCustomAcceptsCustomQuestions 房主排題時要選得到後台新增的題目
func TestBuildCustomAcceptsCustomQuestions(t *testing.T) {
	store := newStore(t)
	svc := services.NewQuestionService(store)

	added, _ := store.Add("後台加的題目", models.ModeSolo, "", 1)
	builtin := services.AllQuestions(models.ModeSolo)[0]

	got := svc.BuildCustom(models.ModeSolo, []int{added.ID, builtin.ID}, nil)
	if len(got) != 2 {
		t.Fatalf("應該組出 2 題，實際 %d", len(got))
	}
	if got[0].ID != added.ID || got[1].ID != builtin.ID {
		t.Fatalf("順序應該照傳入的來，實際 %d, %d", got[0].ID, got[1].ID)
	}
}

// TestBuildRandomSurvivesEverythingDisabled 全部停用時仍要開得起遊戲
func TestBuildRandomSurvivesEverythingDisabled(t *testing.T) {
	store := newStore(t)
	svc := services.NewQuestionService(store)

	for _, q := range services.AllQuestions(models.ModeGroup) {
		_ = store.SetDisabled(q.ID, true)
	}

	got := svc.BuildRandom(models.ModeGroup, 3, "mixed")
	if len(got) == 0 {
		t.Fatal("題目全被停用時仍應該回傳至少一題，否則房間開不起來")
	}
}
