package examples

// 團體題（ID 2001~2025）的示範站位圖。
// 只做團體題：單人題看文字就懂，團體題難的是「站位」，才需要圖。

// baseY 是所有人站立的基準線；viewBox 是 480x270
const baseY = 214.0

// row 產生一排間距平均的人
func row(count int, spacing, y, scale float64, arms ArmPose, legs LegPose, face FacePose) []Figure {
	figures := make([]Figure, 0, count)
	start := -float64(count-1) * spacing / 2

	for i := 0; i < count; i++ {
		figures = append(figures, Figure{
			X:     240 + start + float64(i)*spacing,
			Y:     y,
			Scale: scale,
			Arms:  arms,
			Legs:  legs,
			Face:  face,
		})
	}
	return figures
}

func merge(groups ...[]Figure) []Figure {
	out := make([]Figure, 0)
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

var scenes = map[int]Scene{
	2001: {
		Caption: "全體比出同一個手勢",
		Note:    "四個人、同一個手勢、同一個高度",
		Figures: row(4, 96, baseY, 1, ArmsFinger, LegsStand, FaceSmile),
	},
	2002: {
		Caption: "僵硬的全家福",
		Note:    "肩併肩、手貼身體、笑得越僵越好",
		Figures: row(4, 78, baseY, 1, ArmsDown, LegsStand, FaceSmile),
	},
	2003: {
		Caption: "全部人看同一個方向",
		Note:    "身體朝前，頭跟手一起轉向同一邊",
		Figures: func() []Figure {
			figs := row(4, 96, baseY, 1, ArmsPointR, LegsStand, FaceNeutral)
			for i := range figs {
				figs[i].Tilt = 8
			}
			return figs
		}(),
	},
	2004: {
		Caption: "畢業紀念冊合照",
		Note:    "後排站、前排蹲，臉不要被擋住",
		Figures: merge(
			row(4, 104, baseY-40, 0.86, ArmsDown, LegsStand, FaceSmile),
			row(3, 126, baseY+6, 0.74, ArmsDown, LegsKneel, FaceSmile),
		),
	},
	2005: {
		Caption: "每個人拿同顏色的東西",
		Note:    "四樣同色物品都要清楚入鏡",
		Figures: func() []Figure {
			figs := row(4, 104, baseY, 1, ArmsSide, LegsStand, FaceSmile)
			for i := range figs {
				figs[i].Prop = "◼"
			}
			return figs
		}(),
	},
	2006: {
		Caption: "用身體排出一個英文字母",
		Note:    "每個人是字母的一筆，站遠一點才看得出形狀",
		Figures: []Figure{
			{X: 110, Y: baseY, Scale: 1, Arms: ArmsUp, Legs: LegsWide, Face: FaceSmile},
			{X: 200, Y: baseY, Scale: 1, Arms: ArmsSide, Legs: LegsStand, Face: FaceSmile},
			{X: 290, Y: baseY, Scale: 1, Arms: ArmsUp, Legs: LegsWide, Face: FaceSmile},
			{X: 375, Y: baseY, Scale: 1, Arms: ArmsOneUp, Legs: LegsStand, Face: FaceSmile},
		},
	},
	2007: {
		Caption: "全體跳起來的瞬間",
		Note:    "喊 3、2、1 一起跳，連拍比較好抓",
		Figures: func() []Figure {
			figs := row(4, 100, baseY-22, 1, ArmsUp, LegsJump, FaceLaugh)
			for i := range figs {
				figs[i].Y = baseY - 18 - float64(i%2)*10
			}
			return figs
		}(),
	},
	2008: {
		Caption: "高低三層人像",
		Note:    "站、半蹲、蹲下三層，注意安全不要真的疊人",
		Figures: merge(
			row(2, 150, baseY-56, 0.8, ArmsUp, LegsStand, FaceLaugh),
			row(3, 130, baseY-26, 0.8, ArmsSide, LegsStand, FaceSmile),
			row(2, 170, baseY+8, 0.72, ArmsDown, LegsKneel, FaceSmile),
		),
	},
	2009: {
		Caption: "全體比一個超大的愛心",
		Note:    "雙手在頭上合成愛心，四個人排整齊",
		Figures: row(4, 100, baseY, 1, ArmsHeart, LegsStand, FaceSmile),
	},
	2010: {
		Caption: "樂團封面合照",
		Note:    "高低錯開、表情不要笑，越裝越好",
		Figures: []Figure{
			{X: 100, Y: baseY, Scale: 1, Arms: ArmsSide, Legs: LegsWide, Face: FaceNeutral},
			{X: 205, Y: baseY - 30, Scale: 0.95, Arms: ArmsDown, Legs: LegsStand, Face: FaceNeutral},
			{X: 300, Y: baseY, Scale: 1, Arms: ArmsHug, Legs: LegsSit, Face: FaceNeutral},
			{X: 395, Y: baseY, Scale: 1, Arms: ArmsOneUp, Legs: LegsStand, Face: FaceNeutral},
		},
	},
	2011: {
		Caption: "全體同一個誇張表情",
		Note:    "表情要一致，越浮誇分數越高",
		Figures: row(4, 96, baseY, 1, ArmsSide, LegsWide, FaceLaugh),
	},
	2012: {
		Caption: "被同一件事嚇到",
		Note:    "雙手捧臉、嘴巴張開，全部看同一個點",
		Figures: row(4, 96, baseY, 1, ArmsCheeks, LegsWide, FaceShock),
	},
	2013: {
		Caption: "所有人都在吃東西",
		Note:    "每個人手上都要有食物，而且正在吃",
		Figures: func() []Figure {
			figs := row(4, 104, baseY, 1, ArmsCheeks, LegsStand, FaceLaugh)
			for i := range figs {
				figs[i].Prop = "🍩"
			}
			return figs
		}(),
	},
	2014: {
		Caption: "排成一直線只露一隻眼",
		Note:    "前後排成一排，後面的人只探出半張臉",
		Figures: []Figure{
			{X: 300, Y: baseY, Scale: 1.1, Arms: ArmsDown, Legs: LegsStand, Face: FaceNeutral},
			{X: 262, Y: baseY - 6, Scale: 1.02, Arms: ArmsDown, Legs: LegsStand, Face: FaceNeutral},
			{X: 226, Y: baseY - 12, Scale: 0.94, Arms: ArmsDown, Legs: LegsStand, Face: FaceNeutral},
			{X: 192, Y: baseY - 18, Scale: 0.86, Arms: ArmsDown, Legs: LegsStand, Face: FaceNeutral},
		},
	},
	2015: {
		Caption: "犯罪現場擺拍",
		Note:    "一個人躺平，其他人圍著做誇張反應",
		Figures: []Figure{
			{X: 170, Y: baseY - 4, Scale: 0.95, Arms: ArmsUp, Legs: LegsWide, Face: FaceNeutral, Tilt: 90},
			{X: 285, Y: baseY, Scale: 1, Arms: ArmsCheeks, Legs: LegsStand, Face: FaceShock},
			{X: 360, Y: baseY, Scale: 1, Arms: ArmsPointL, Legs: LegsStand, Face: FaceShock},
			{X: 425, Y: baseY, Scale: 0.95, Arms: ArmsDown, Legs: LegsKneel, Face: FaceShock},
		},
	},
	2016: {
		Caption: "全體背對鏡頭",
		Note:    "只用背影和手勢表達開心",
		Figures: row(4, 96, baseY, 1, ArmsUp, LegsStand, FaceBack),
	},
	2017: {
		Caption: "每人比一根手指湊成數字",
		Note:    "手指要都在畫面裡，加起來剛好是那個數字",
		Figures: row(4, 96, baseY, 1, ArmsFinger, LegsStand, FaceSmile),
	},
	2018: {
		Caption: "最擠的合照",
		Note:    "越擠越好，人頭數就是分數",
		Figures: func() []Figure {
			figs := row(6, 62, baseY, 0.92, ArmsHug, LegsStand, FaceLaugh)
			for i := range figs {
				figs[i].Y = baseY - float64(i%2)*12
			}
			return figs
		}(),
	},
	2019: {
		Caption: "合力抬起一個東西",
		Note:    "所有人的手都要在同一個高度、同一個方向使力",
		Figures: row(4, 96, baseY, 1, ArmsLift, LegsWide, FaceNeutral),
	},
	2020: {
		Caption: "全體模仿同一隻動物",
		Note:    "四肢著地或蹲低，動作要一致",
		Figures: row(4, 100, baseY, 1, ArmsDown, LegsKneel, FaceLaugh),
	},
	2021: {
		Caption: "錯位：有人被放在手掌上",
		Note:    "近的人伸出手，遠的人站到手掌的位置上，鏡頭要對齊",
		Figures: []Figure{
			// 小人的落腳點必須剛好是大人手掌的末端，否則看起來只是浮在半空。
			// 手掌末端 = X + 36*Scale, Y + (shoulderY-12)*Scale
			{X: 112, Y: baseY + 30, Scale: 1.55, Arms: ArmsPointR, Legs: LegsWide, Face: FaceSmile},
			{X: 168, Y: baseY - 82, Scale: 0.52, Arms: ArmsUp, Legs: LegsStand, Face: FaceLaugh},
		},
	},
	2022: {
		Caption: "全體舉手，手勢都不一樣",
		Note:    "每個人的手勢都要不同，一眼看得出差別",
		Figures: []Figure{
			{X: 105, Y: baseY, Scale: 1, Arms: ArmsUp, Legs: LegsStand, Face: FaceSmile},
			{X: 200, Y: baseY, Scale: 1, Arms: ArmsFinger, Legs: LegsStand, Face: FaceSmile},
			{X: 295, Y: baseY, Scale: 1, Arms: ArmsOneUp, Legs: LegsStand, Face: FaceSmile},
			{X: 388, Y: baseY, Scale: 1, Arms: ArmsHeart, Legs: LegsStand, Face: FaceSmile},
		},
	},
	2023: {
		Caption: "高級感廣告合照",
		Note:    "站姿放鬆、不要看鏡頭、留白多一點",
		Figures: []Figure{
			{X: 130, Y: baseY, Scale: 1, Arms: ArmsHug, Legs: LegsStand, Face: FaceNeutral, Tilt: -4},
			{X: 235, Y: baseY, Scale: 1, Arms: ArmsSide, Legs: LegsSit, Face: FaceNeutral},
			{X: 350, Y: baseY, Scale: 1, Arms: ArmsDown, Legs: LegsWide, Face: FaceNeutral, Tilt: 4},
		},
	},
	2024: {
		Caption: "一起指向畫面外同一點",
		Note:    "手臂伸直、指同一個方向，眼睛也要跟著看",
		Figures: row(4, 96, baseY, 1, ArmsPointR, LegsStand, FaceSmile),
	},
	2025: {
		Caption: "大笑但沒人看鏡頭",
		Note:    "互相看著笑，就是不要看鏡頭",
		Figures: []Figure{
			{X: 110, Y: baseY, Scale: 1, Arms: ArmsHug, Legs: LegsStand, Face: FaceLaugh, Tilt: 10},
			{X: 210, Y: baseY, Scale: 1, Arms: ArmsCheeks, Legs: LegsStand, Face: FaceLaugh, Tilt: -12},
			{X: 310, Y: baseY, Scale: 1, Arms: ArmsDown, Legs: LegsStand, Face: FaceLaugh, Tilt: 14},
			{X: 400, Y: baseY, Scale: 1, Arms: ArmsOneUp, Legs: LegsStand, Face: FaceLaugh, Tilt: -8},
		},
	},
}

// Builtin 取得內建示範圖的 SVG；沒有就回傳 false
func Builtin(questionID int) (string, bool) {
	scene, ok := scenes[questionID]
	if !ok {
		return "", false
	}
	return Render(scene), true
}

// HasBuiltin 是否有內建示範圖
func HasBuiltin(questionID int) bool {
	_, ok := scenes[questionID]
	return ok
}

// BuiltinIDs 有內建示範圖的題目 ID
func BuiltinIDs() []int {
	ids := make([]int, 0, len(scenes))
	for id := range scenes {
		ids = append(ids, id)
	}
	return ids
}
