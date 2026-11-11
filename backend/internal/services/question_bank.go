package services

import "beauty-game/internal/models"

// soloQuestions 單人題庫（72 題）。ID 1000 起算。
// Difficulty：1 = 基礎（拍了就有）、2 = 進階（要找東西、要構圖）
var soloQuestions = []models.Question{
	// ── 表情 expression（10）────────────────────────────────
	{ID: 1001, Text: "拍一張「剛中樂透」的表情", Category: models.CategoryExpression, Difficulty: 1},
	{ID: 1002, Text: "拍出你這輩子最兇的臉", Category: models.CategoryExpression, Difficulty: 1},
	{ID: 1003, Text: "假裝你聞到了世界上最臭的味道", Category: models.CategoryExpression, Difficulty: 1},
	{ID: 1004, Text: "裝出「已經一個禮拜沒睡」的死魚眼", Category: models.CategoryExpression, Difficulty: 1},
	{ID: 1005, Text: "拍一張假哭到梨花帶淚的臉", Category: models.CategoryExpression, Difficulty: 1},
	{ID: 1013, Text: "打開冰箱發現點心被吃光的表情", Category: models.CategoryExpression, Difficulty: 1},
	{ID: 1007, Text: "露出「我全都知道了」的邪魅微笑", Category: models.CategoryExpression, Difficulty: 1},
	{ID: 1011, Text: "吃到超酸檸檬的表情", Category: models.CategoryExpression, Difficulty: 1},
	{ID: 1012, Text: "偶像本人突然站在你面前的表情", Category: models.CategoryExpression, Difficulty: 1},
	{ID: 1010, Text: "拍出你看到帳單那一秒的表情", Category: models.CategoryExpression, Difficulty: 1},

	// ── 模仿 imitate（5）────────────────────────────────────
	{ID: 1103, Text: "模仿一座希臘雕像，要有肌肉線條", Category: models.CategoryImitate, Difficulty: 1},
	{ID: 1110, Text: "模仿一隻被叫到名字、歪頭的狗狗", Category: models.CategoryImitate, Difficulty: 1},
	{ID: 1111, Text: "模仿拿著香蕉講電話的阿嬤", Category: models.CategoryImitate, Difficulty: 1},
	{ID: 1106, Text: "模仿超浮誇 YouTuber 的震驚封面臉", Category: models.CategoryImitate, Difficulty: 1},
	{ID: 1108, Text: "模仿手機只剩 1% 電的人", Category: models.CategoryImitate, Difficulty: 2},

	// ── 自拍姿勢 pose（7）──────────────────────────────────
	{ID: 1201, Text: "45 度角網美自拍，要有下巴線條", Category: models.CategoryPose, Difficulty: 1},
	{ID: 1209, Text: "比出愛心，但臉要超級臭", Category: models.CategoryPose, Difficulty: 1},
	{ID: 1203, Text: "單手托腮的憂鬱文青風自拍", Category: models.CategoryPose, Difficulty: 1},
	{ID: 1204, Text: "拍一張擠出雙下巴的仰角自拍", Category: models.CategoryPose, Difficulty: 1},
	{ID: 1205, Text: "用手指框住自己的臉當相框", Category: models.CategoryPose, Difficulty: 1},
	{ID: 1210, Text: "假裝在拍偶像劇，來一個深情回眸", Category: models.CategoryPose, Difficulty: 1},
	{ID: 1207, Text: "拍一張假裝被偷拍的明星街拍照", Category: models.CategoryPose, Difficulty: 2},

	// ── 找顏色 color（5）──────────────────────────────────
	{ID: 1301, Text: "把一個「紅色」的東西貼在臉上自拍", Category: models.CategoryColor, Difficulty: 1},
	{ID: 1302, Text: "找到一個「藍色」的東西頂在頭上", Category: models.CategoryColor, Difficulty: 1},
	{ID: 1303, Text: "找到一個「黃色」的東西遮住半張臉", Category: models.CategoryColor, Difficulty: 1},
	{ID: 1304, Text: "找到一個「綠色」的東西當項鍊掛著", Category: models.CategoryColor, Difficulty: 1},
	{ID: 1305, Text: "跟身邊顏色最鮮豔的東西深情對望", Category: models.CategoryColor, Difficulty: 2},

	// ── 找物品 object（5）──────────────────────────────────
	{ID: 1401, Text: "找一個圓形的東西戴在頭上", Category: models.CategoryObject, Difficulty: 1},
	{ID: 1402, Text: "找一支筆當麥克風，擺出唱歌的樣子", Category: models.CategoryObject, Difficulty: 1},
	{ID: 1403, Text: "隨便拿個東西當紅酒杯搖晃耍帥", Category: models.CategoryObject, Difficulty: 1},
	{ID: 1405, Text: "隨便拿一樣東西當武器，擺出殺氣臉", Category: models.CategoryObject, Difficulty: 1},
	{ID: 1409, Text: "拿一個包包，假裝是剛買的名牌包在炫耀", Category: models.CategoryObject, Difficulty: 1},

	// ── 學生篇 student（11）──────────────────────────────
	{ID: 1601, Text: "考卷發下來，看到「38 分」那一秒的表情", Category: models.CategoryStudent, Difficulty: 1},
	{ID: 1602, Text: "成績單全部歐趴，忍不住偷笑的表情", Category: models.CategoryStudent, Difficulty: 1},
	{ID: 1603, Text: "上課遲到，從後門躡手躡腳溜進教室的樣子", Category: models.CategoryStudent, Difficulty: 1},
	{ID: 1604, Text: "上課打瞌睡被點名，瞬間驚醒的臉", Category: models.CategoryStudent, Difficulty: 1},
	{ID: 1605, Text: "考前一晚才第一次翻開課本的絕望表情", Category: models.CategoryStudent, Difficulty: 1},
	{ID: 1606, Text: "老師突然宣布「現在隨堂考」的表情", Category: models.CategoryStudent, Difficulty: 1},
	{ID: 1612, Text: "下課鐘一響，衝去福利社的樣子", Category: models.CategoryStudent, Difficulty: 1},
	{ID: 1613, Text: "上台報告講到一半突然忘詞的表情", Category: models.CategoryStudent, Difficulty: 1},
	{ID: 1609, Text: "分組報告，組員全部已讀不回的表情", Category: models.CategoryStudent, Difficulty: 1},
	{ID: 1610, Text: "偷吃早餐被老師發現，嘴巴還塞滿的樣子", Category: models.CategoryStudent, Difficulty: 1},
	{ID: 1611, Text: "半夜玩手遊被媽媽發現的表情", Category: models.CategoryStudent, Difficulty: 1},

	// ── 職場篇 work（10）─────────────────────────────────
	{ID: 1711, Text: "客戶說「再改一下就好」的表情", Category: models.CategoryWork, Difficulty: 1},
	{ID: 1702, Text: "明明是老闆的錯，最後卻怪到你頭上的表情", Category: models.CategoryWork, Difficulty: 1},
	{ID: 1703, Text: "翹班出去買飲料，被主管當場抓到的表情", Category: models.CategoryWork, Difficulty: 1},
	{ID: 1704, Text: "週一早上打卡前，靈魂出竅的臉", Category: models.CategoryWork, Difficulty: 1},
	{ID: 1705, Text: "下班前五分鐘，老闆說「我們來開個會」", Category: models.CategoryWork, Difficulty: 1},
	{ID: 1706, Text: "薪水入帳那一秒的表情", Category: models.CategoryWork, Difficulty: 1},
	{ID: 1707, Text: "年終獎金只有預期一半的表情", Category: models.CategoryWork, Difficulty: 1},
	{ID: 1708, Text: "假裝聽得懂老闆在講什麼的表情", Category: models.CategoryWork, Difficulty: 1},
	{ID: 1709, Text: "同事說「幫我一下，只要五分鐘」的表情", Category: models.CategoryWork, Difficulty: 1},
	{ID: 1710, Text: "收到「明天放颱風假」通知的表情", Category: models.CategoryWork, Difficulty: 1},

	// ── 教會 church（9）────────────────────────
	{ID: 1801, Text: "打開手機訊息，發現新朋友放你鳥的表情", Category: models.CategoryChurch, Difficulty: 1},
	{ID: 1803, Text: "聚會遲到推門進來，全場回頭看你的表情", Category: models.CategoryChurch, Difficulty: 1},
	{ID: 1804, Text: "想吃最後一塊炸雞，卻看到新朋友也想吃時的反應", Category: models.CategoryChurch, Difficulty: 1},
	{ID: 1805, Text: "交換禮物拆到很雷的東西，還要假裝很喜歡", Category: models.CategoryChurch, Difficulty: 1},
	{ID: 1806, Text: "敬拜唱到最高音，快破音的表情", Category: models.CategoryChurch, Difficulty: 1},
	{ID: 1807, Text: "收到通知「今年聖誕戲劇由你當主角」的表情", Category: models.CategoryChurch, Difficulty: 1},
	{ID: 1808, Text: "聚會延長半小時，很想上廁所的表情", Category: models.CategoryChurch, Difficulty: 1},
	{ID: 1809, Text: "尷尬而不失禮貌的微笑", Category: models.CategoryChurch, Difficulty: 1},
	{ID: 1810, Text: "聚會偷溜，在門口剛好被牧師看到的表情", Category: models.CategoryChurch, Difficulty: 1},

	// ── 聖誕市集 christmas（10）──────────────────────────
	{ID: 1901, Text: "賣鬆餅時，客人問「有番茄醬嗎？」的表情", Category: models.CategoryChristmas, Difficulty: 1},
	{ID: 1902, Text: "問客人「要全糖嗎？」，客人回「加一點辣」的表情", Category: models.CategoryChristmas, Difficulty: 1},
	{ID: 1903, Text: "客人吃了一口說「不好吃，可以退費嗎？」的表情", Category: models.CategoryChristmas, Difficulty: 1},
	{ID: 1904, Text: "二手衣攤被問「這件可以打 3 折嗎？」的表情", Category: models.CategoryChristmas, Difficulty: 1},
	{ID: 1905, Text: "顧攤一整天，終於有客人走過來的表情", Category: models.CategoryChristmas, Difficulty: 1},
	{ID: 1906, Text: "客人拿 1000 元，只買一個 20 元的東西", Category: models.CategoryChristmas, Difficulty: 1},
	{ID: 1907, Text: "隔壁攤大排長龍，你的攤位沒半個人", Category: models.CategoryChristmas, Difficulty: 1},
	{ID: 1908, Text: "客人問「這是你自己做的嗎？」其實是買來的表情", Category: models.CategoryChristmas, Difficulty: 1},
	{ID: 1909, Text: "戴上聖誕帽，用最熱情的笑容大聲叫賣", Category: models.CategoryChristmas, Difficulty: 1},
	{ID: 1910, Text: "收攤數錢，發現少了 100 元的表情", Category: models.CategoryChristmas, Difficulty: 1},
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
