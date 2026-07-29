package services

import "beauty-game/internal/models"

// soloQuestions 單人題庫（50 題）。ID 1000 起算。
// Difficulty：1 = 基礎（拍了就有）、2 = 進階（要找東西、要構圖）
var soloQuestions = []models.Question{
	// ── 表情 expression（10）────────────────────────────────
	{ID: 1001, Text: "拍一張「剛中樂透」的表情", Category: models.CategoryExpression, Difficulty: 1},
	{ID: 1002, Text: "拍出你這輩子最兇的臉", Category: models.CategoryExpression, Difficulty: 1},
	{ID: 1003, Text: "假裝你聞到了世界上最臭的味道", Category: models.CategoryExpression, Difficulty: 1},
	{ID: 1004, Text: "裝出「已經一個禮拜沒睡」的死魚眼", Category: models.CategoryExpression, Difficulty: 1},
	{ID: 1005, Text: "拍一張假哭到梨花帶淚的臉", Category: models.CategoryExpression, Difficulty: 1},
	{ID: 1006, Text: "做出被雷打到的驚嚇表情", Category: models.CategoryExpression, Difficulty: 1},
	{ID: 1007, Text: "露出「我全都知道了」的邪魅微笑", Category: models.CategoryExpression, Difficulty: 1},
	{ID: 1008, Text: "拍一張只有左半邊臉在笑的照片", Category: models.CategoryExpression, Difficulty: 2},
	{ID: 1009, Text: "同時做到「挑一邊眉毛」+「嘟嘴」", Category: models.CategoryExpression, Difficulty: 2},
	{ID: 1010, Text: "拍出你看到帳單那一秒的表情", Category: models.CategoryExpression, Difficulty: 1},

	// ── 模仿 imitate（8）────────────────────────────────────
	{ID: 1101, Text: "模仿一隻睡到翻肚的貓", Category: models.CategoryImitate, Difficulty: 1},
	{ID: 1102, Text: "模仿你老闆或老師講話的樣子", Category: models.CategoryImitate, Difficulty: 1},
	{ID: 1103, Text: "模仿一座希臘雕像，要有肌肉線條", Category: models.CategoryImitate, Difficulty: 1},
	{ID: 1104, Text: "模仿電梯裡尷尬看樓層的人", Category: models.CategoryImitate, Difficulty: 1},
	{ID: 1105, Text: "模仿一株快枯掉的植物", Category: models.CategoryImitate, Difficulty: 1},
	{ID: 1106, Text: "模仿超商店員說「內用還是外帶」", Category: models.CategoryImitate, Difficulty: 1},
	{ID: 1107, Text: "模仿一隻正在偷東西的浣熊", Category: models.CategoryImitate, Difficulty: 2},
	{ID: 1108, Text: "模仿手機只剩 1% 電的人", Category: models.CategoryImitate, Difficulty: 2},

	// ── 自拍姿勢 pose（8）──────────────────────────────────
	{ID: 1201, Text: "45 度角網美自拍，要有下巴線條", Category: models.CategoryPose, Difficulty: 1},
	{ID: 1202, Text: "比出愛心，而且全程不能笑", Category: models.CategoryPose, Difficulty: 1},
	{ID: 1203, Text: "單手托腮的憂鬱文青風自拍", Category: models.CategoryPose, Difficulty: 1},
	{ID: 1204, Text: "拍一張只露出眼睛以上的自拍", Category: models.CategoryPose, Difficulty: 1},
	{ID: 1205, Text: "用手指框住自己的臉當相框", Category: models.CategoryPose, Difficulty: 1},
	{ID: 1206, Text: "拍一張從下往上的超級醜角度", Category: models.CategoryPose, Difficulty: 1},
	{ID: 1207, Text: "拍一張看起來像在飛的照片", Category: models.CategoryPose, Difficulty: 2},
	{ID: 1208, Text: "用「手在畫面前、臉在後面」拍出景深", Category: models.CategoryPose, Difficulty: 2},

	// ── 找顏色 color（8）──────────────────────────────────
	{ID: 1301, Text: "找到一個「紅色」的東西並入鏡", Category: models.CategoryColor, Difficulty: 1},
	{ID: 1302, Text: "找到一個「藍色」的東西頂在頭上", Category: models.CategoryColor, Difficulty: 1},
	{ID: 1303, Text: "找到一個「黃色」的東西遮住半張臉", Category: models.CategoryColor, Difficulty: 1},
	{ID: 1304, Text: "找到一個「綠色」的東西當項鍊掛著", Category: models.CategoryColor, Difficulty: 1},
	{ID: 1305, Text: "拍一張畫面裡完全沒有白色的照片", Category: models.CategoryColor, Difficulty: 2},
	{ID: 1306, Text: "把三種不同顏色的東西排成一排拍下來", Category: models.CategoryColor, Difficulty: 2},
	{ID: 1307, Text: "找到跟你今天衣服同色的東西合照", Category: models.CategoryColor, Difficulty: 2},
	{ID: 1308, Text: "拍一張畫面裡有彩虹四種以上顏色的照片", Category: models.CategoryColor, Difficulty: 2},

	// ── 找物品 object（8）──────────────────────────────────
	{ID: 1401, Text: "找一個圓形的東西戴在頭上", Category: models.CategoryObject, Difficulty: 1},
	{ID: 1402, Text: "找一支筆當麥克風，擺出唱歌的樣子", Category: models.CategoryObject, Difficulty: 1},
	{ID: 1403, Text: "找一個杯子，假裝在喝下午茶", Category: models.CategoryObject, Difficulty: 1},
	{ID: 1404, Text: "找一條線狀的東西當鬍子", Category: models.CategoryObject, Difficulty: 1},
	{ID: 1405, Text: "找出你身邊最舊的一樣東西", Category: models.CategoryObject, Difficulty: 1},
	{ID: 1406, Text: "找一個東西擋住臉，只露出一隻眼睛", Category: models.CategoryObject, Difficulty: 1},
	{ID: 1407, Text: "找兩個東西組成一張笑臉", Category: models.CategoryObject, Difficulty: 2},
	{ID: 1408, Text: "找一個透明的東西，透過它拍自己", Category: models.CategoryObject, Difficulty: 2},

	// ── 進階複合 advanced（8）──────────────────────────────
	{ID: 1501, Text: "在白色背景前，拍一張只有黑白兩色的自拍", Category: models.CategoryAdvanced, Difficulty: 2},
	{ID: 1502, Text: "拿著綠色的東西，在窗邊逆光自拍", Category: models.CategoryAdvanced, Difficulty: 2},
	{ID: 1503, Text: "在鏡子或任何反射面前拍一張倒影自拍", Category: models.CategoryAdvanced, Difficulty: 2},
	{ID: 1504, Text: "找一個門框或窗框，把自己框在裡面", Category: models.CategoryAdvanced, Difficulty: 2},
	{ID: 1505, Text: "拍一張有影子入鏡、而且影子是主角的照片", Category: models.CategoryAdvanced, Difficulty: 2},
	{ID: 1506, Text: "在最暗的角落拍一張只靠手機螢幕打光的臉", Category: models.CategoryAdvanced, Difficulty: 2},
	{ID: 1507, Text: "用一個物品製造錯位，讓它看起來被你捏在手上", Category: models.CategoryAdvanced, Difficulty: 2},
	{ID: 1508, Text: "在有文字的背景前自拍，讓文字剛好在你頭上", Category: models.CategoryAdvanced, Difficulty: 2},
}

// groupQuestions 團體題庫（25 題）。ID 2000 起算，全部都是合體照。
var groupQuestions = []models.Question{
	{ID: 2001, Text: "全體比出同一個手勢，一個都不能歪", Category: models.CategoryGroup, Difficulty: 1},
	{ID: 2002, Text: "拍一張大家都在假笑的僵硬全家福", Category: models.CategoryGroup, Difficulty: 1},
	{ID: 2003, Text: "全部人看向同一個奇怪的方向", Category: models.CategoryGroup, Difficulty: 1},
	{ID: 2004, Text: "模仿一張畢業紀念冊的合照", Category: models.CategoryGroup, Difficulty: 1},
	{ID: 2005, Text: "每個人拿一樣「同顏色」的東西合照", Category: models.CategoryGroup, Difficulty: 2},
	{ID: 2006, Text: "全體用身體排出一個英文字母", Category: models.CategoryGroup, Difficulty: 2},
	{ID: 2007, Text: "拍一張大家都在跳起來的瞬間", Category: models.CategoryGroup, Difficulty: 2},
	{ID: 2008, Text: "疊出高低排列的三層人像（請注意安全）", Category: models.CategoryGroup, Difficulty: 2},
	{ID: 2009, Text: "全體一起比出一個超大的愛心", Category: models.CategoryGroup, Difficulty: 1},
	{ID: 2010, Text: "拍一張看起來像樂團封面的合照", Category: models.CategoryGroup, Difficulty: 2},
	{ID: 2011, Text: "全體做出同一個表情，越誇張越好", Category: models.CategoryGroup, Difficulty: 1},
	{ID: 2012, Text: "假裝大家正在被同一件事嚇到", Category: models.CategoryGroup, Difficulty: 1},
	{ID: 2013, Text: "拍一張所有人都在吃東西的合照", Category: models.CategoryGroup, Difficulty: 1},
	{ID: 2014, Text: "排成一直線，只露出每個人的一隻眼睛", Category: models.CategoryGroup, Difficulty: 2},
	{ID: 2015, Text: "拍一張像犯罪現場的搞笑擺拍", Category: models.CategoryGroup, Difficulty: 2},
	{ID: 2016, Text: "全體背對鏡頭，只用背影表達開心", Category: models.CategoryGroup, Difficulty: 2},
	{ID: 2017, Text: "每個人比一根手指，合起來變成一個數字", Category: models.CategoryGroup, Difficulty: 2},
	{ID: 2018, Text: "拍一張最擠的合照，塞越多人越好", Category: models.CategoryGroup, Difficulty: 1},
	{ID: 2019, Text: "假裝大家正在合力抬起一個東西", Category: models.CategoryGroup, Difficulty: 1},
	{ID: 2020, Text: "全體模仿同一隻動物", Category: models.CategoryGroup, Difficulty: 1},
	{ID: 2021, Text: "拍一張有人被「錯位」放在手掌上的合照", Category: models.CategoryGroup, Difficulty: 2},
	{ID: 2022, Text: "全體舉手，但每個人的手勢都不一樣", Category: models.CategoryGroup, Difficulty: 1},
	{ID: 2023, Text: "拍一張假裝在拍廣告的高級感合照", Category: models.CategoryGroup, Difficulty: 2},
	{ID: 2024, Text: "全部人一起指向畫面外的同一點", Category: models.CategoryGroup, Difficulty: 1},
	{ID: 2025, Text: "拍一張大家都在大笑、沒有人在看鏡頭的照片", Category: models.CategoryGroup, Difficulty: 2},
}

// init 補上題庫每題的 Mode，避免上面手寫時漏填
func init() {
	for i := range soloQuestions {
		soloQuestions[i].Mode = models.ModeSolo
	}
	for i := range groupQuestions {
		groupQuestions[i].Mode = models.ModeGroup
	}
}

// AllQuestions 依模式取得完整題庫（給前端題目選擇器用）
func AllQuestions(mode models.GameMode) []models.Question {
	src := soloQuestions
	if mode == models.ModeGroup {
		src = groupQuestions
	}

	out := make([]models.Question, len(src))
	copy(out, src)
	return out
}
