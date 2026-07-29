// Package examples 產生團體題的示範站位圖。
//
// 這些圖是程式畫的火柴人，不是外部素材：內嵌在執行檔裡、不外連、沒有版權問題，
// 而且「站位」這種資訊用線條圖比真人照片更好讀。房主可以在後台上傳自己的圖蓋掉。
package examples

import (
	"fmt"
	"math"
	"strings"
)

// 單一火柴人在自身座標系裡的尺寸（原點在雙腳中間）
const (
	headR      = 13.0
	headY      = -80.0
	neckY      = -67.0
	shoulderY  = -60.0
	hipY       = -30.0
	armLen     = 26.0
	legSpreadX = 13.0
)

// ArmPose 手的姿勢
type ArmPose string

const (
	ArmsDown   ArmPose = "down"
	ArmsUp     ArmPose = "up"
	ArmsSide   ArmPose = "side"
	ArmsHeart  ArmPose = "heart"  // 雙手在頭上比愛心
	ArmsPointR ArmPose = "pointR" // 指向右方
	ArmsPointL ArmPose = "pointL"
	ArmsCheeks ArmPose = "cheeks" // 雙手捧臉（驚嚇）
	ArmsHug    ArmPose = "hug"    // 手往內收（擠在一起）
	ArmsOneUp  ArmPose = "oneUp"  // 一手舉高一手放下
	ArmsLift   ArmPose = "lift"   // 雙手往前平抬（合力抬東西）
	ArmsFinger ArmPose = "finger" // 單手往上比一根手指
)

// LegPose 腳的姿勢
type LegPose string

const (
	LegsStand LegPose = "stand"
	LegsJump  LegPose = "jump"
	LegsWide  LegPose = "wide"
	LegsKneel LegPose = "kneel"
	LegsSit   LegPose = "sit"
)

// FacePose 臉
type FacePose string

const (
	FaceSmile   FacePose = "smile"
	FaceLaugh   FacePose = "laugh"
	FaceShock   FacePose = "shock"
	FaceNeutral FacePose = "neutral"
	FaceBack    FacePose = "back" // 背對鏡頭，不畫五官
)

// Figure 一個人
type Figure struct {
	X, Y  float64 // 雙腳落點
	Scale float64
	Arms  ArmPose
	Legs  LegPose
	Face  FacePose
	Tilt  float64 // 整個人傾斜幾度
	Prop  string  // 手上拿的東西，畫成一個小方塊 + 文字
}

// Scene 一張示範圖
type Scene struct {
	Caption string
	Figures []Figure
	// Note 疊在圖上的補充提示（例如「由下往上拍」）
	Note string
}

func f(v float64) string {
	return fmt.Sprintf("%.1f", v)
}

func line(x1, y1, x2, y2 float64) string {
	return `<line x1="` + f(x1) + `" y1="` + f(y1) + `" x2="` + f(x2) + `" y2="` + f(y2) + `"/>`
}

// poly 用折線畫肢體，中間的轉折點就是手肘或膝蓋。
// 直線畫出來的手很難分辨姿勢，有關節才看得懂在做什麼。
func poly(points ...[2]float64) string {
	parts := make([]string, 0, len(points))
	for _, p := range points {
		parts = append(parts, f(p[0])+","+f(p[1]))
	}
	return `<polyline points="` + strings.Join(parts, " ") + `"/>`
}

// mirrored 把一組點左右鏡射，畫對稱姿勢用
func mirrored(points [][2]float64) [][2]float64 {
	out := make([][2]float64, len(points))
	for i, p := range points {
		out[i] = [2]float64{-p[0], p[1]}
	}
	return out
}

// bothArms 畫左右對稱的兩隻手
func bothArms(points ...[2]float64) string {
	return poly(points...) + poly(mirrored(points)...)
}

// arms 依姿勢畫出兩隻手，另外回傳要疊上去的裝飾（愛心、手指）
func arms(pose ArmPose) (string, string) {
	shoulder := [2]float64{0, shoulderY}
	decoration := ""
	body := ""

	switch pose {
	case ArmsUp:
		body = bothArms(shoulder, [2]float64{20, shoulderY - 16}, [2]float64{24, shoulderY - 36})

	case ArmsSide:
		body = bothArms(shoulder, [2]float64{17, shoulderY + 3}, [2]float64{32, shoulderY - 4})

	case ArmsHeart:
		// 手肘往外撐開，手腕再收到頭頂上方，才不會把臉蓋掉
		body = bothArms(shoulder, [2]float64{24, shoulderY - 18}, [2]float64{10, headY - 20})
		decoration = `<path d="M0 ` + f(headY-36) +
			` c -7 -9 -19 -2 -12 7 l 12 13 l 12 -13 c 7 -9 -5 -16 -12 -7 z" fill="#f92e83" stroke="none"/>`

	case ArmsPointR:
		body = poly(shoulder, [2]float64{18, shoulderY - 6}, [2]float64{36, shoulderY - 12}) +
			poly(shoulder, [2]float64{14, shoulderY + 16}, [2]float64{17, hipY - 2})

	case ArmsPointL:
		body = poly(shoulder, [2]float64{-18, shoulderY - 6}, [2]float64{-36, shoulderY - 12}) +
			poly(shoulder, [2]float64{-14, shoulderY + 16}, [2]float64{-17, hipY - 2})

	case ArmsCheeks:
		// 手肘外開、手掌貼在臉頰旁，是「嚇到」的經典動作
		body = bothArms(shoulder, [2]float64{23, shoulderY - 8}, [2]float64{15, headY + 2})

	case ArmsHug:
		body = bothArms(shoulder, [2]float64{15, shoulderY + 14}, [2]float64{3, hipY - 4})

	case ArmsOneUp:
		body = poly(shoulder, [2]float64{-20, shoulderY - 16}, [2]float64{-24, shoulderY - 36}) +
			poly(shoulder, [2]float64{14, shoulderY + 16}, [2]float64{17, hipY - 2})

	case ArmsLift:
		body = bothArms(shoulder, [2]float64{19, shoulderY - 4}, [2]float64{28, shoulderY - 20})

	case ArmsFinger:
		body = poly(shoulder, [2]float64{16, shoulderY - 12}, [2]float64{13, shoulderY - 34}) +
			poly(shoulder, [2]float64{-14, shoulderY + 16}, [2]float64{-17, hipY - 2})
		decoration = `<circle cx="13" cy="` + f(shoulderY-40) + `" r="4" fill="#ffd166" stroke="none"/>`

	default: // ArmsDown
		body = bothArms(shoulder, [2]float64{15, shoulderY + 16}, [2]float64{18, hipY - 2})
	}

	return body, decoration
}

func legs(pose LegPose) string {
	hip := [2]float64{0, hipY}

	switch pose {
	case LegsJump:
		// 腳離地並往後收，腳尖不落在基準線上才有跳起來的感覺
		return poly(hip, [2]float64{-legSpreadX - 10, hipY + 14}, [2]float64{-legSpreadX - 4, hipY + 24}) +
			poly(hip, [2]float64{legSpreadX + 10, hipY + 14}, [2]float64{legSpreadX + 4, hipY + 24})

	case LegsWide:
		return poly(hip, [2]float64{-legSpreadX - 6, hipY + 16}, [2]float64{-legSpreadX - 11, 0}) +
			poly(hip, [2]float64{legSpreadX + 6, hipY + 16}, [2]float64{legSpreadX + 11, 0})

	case LegsKneel:
		// 蹲姿：膝蓋往外開、小腿垂直落地
		return poly(hip, [2]float64{-20, hipY + 12}, [2]float64{-15, 0}) +
			poly(hip, [2]float64{20, hipY + 12}, [2]float64{15, 0})

	case LegsSit:
		// 坐姿：大腿往前、小腿垂直
		return poly(hip, [2]float64{22, hipY + 2}, [2]float64{22, 0}) +
			poly(hip, [2]float64{12, hipY + 6}, [2]float64{10, 0})

	default: // LegsStand
		return poly(hip, [2]float64{-legSpreadX + 2, hipY + 16}, [2]float64{-legSpreadX, 0}) +
			poly(hip, [2]float64{legSpreadX - 2, hipY + 16}, [2]float64{legSpreadX, 0})
	}
}

func face(pose FacePose) string {
	if pose == FaceBack {
		// 背對鏡頭：用一撮頭髮示意
		return `<path d="M-7 ` + f(headY-8) + ` q 7 -6 14 0" fill="none"/>`
	}

	eyes := `<circle cx="-4.5" cy="` + f(headY-3) + `" r="1.6" fill="currentColor" stroke="none"/>` +
		`<circle cx="4.5" cy="` + f(headY-3) + `" r="1.6" fill="currentColor" stroke="none"/>`

	switch pose {
	case FaceLaugh:
		return eyes + `<path d="M-6 ` + f(headY+3) + ` q 6 8 12 0 z" fill="currentColor" stroke="none"/>`
	case FaceShock:
		return `<circle cx="-4.5" cy="` + f(headY-3) + `" r="2.6" fill="currentColor" stroke="none"/>` +
			`<circle cx="4.5" cy="` + f(headY-3) + `" r="2.6" fill="currentColor" stroke="none"/>` +
			`<ellipse cx="0" cy="` + f(headY+5) + `" rx="3.4" ry="4.6" fill="currentColor" stroke="none"/>`
	case FaceNeutral:
		return eyes + line(-5, headY+5, 5, headY+5)
	default: // FaceSmile
		return eyes + `<path d="M-6 ` + f(headY+2) + ` q 6 6 12 0" fill="none"/>`
	}
}

func prop(label string) string {
	if label == "" {
		return ""
	}
	return `<rect x="18" y="` + f(shoulderY-14) + `" width="22" height="16" rx="3" fill="#38bdf8" stroke="none"/>` +
		`<text x="29" y="` + f(shoulderY-2) + `" font-size="10" text-anchor="middle" fill="#0f172a" stroke="none">` +
		label + `</text>`
}

func drawFigure(fig Figure) string {
	if fig.Scale == 0 {
		fig.Scale = 1
	}

	armPath, decoration := arms(fig.Arms)

	body := line(0, neckY, 0, hipY) +
		armPath +
		legs(fig.Legs) +
		// 頭和臉最後畫，壓在肢體上面，臉才不會被手臂穿過去
		`<circle cx="0" cy="` + f(headY) + `" r="` + f(headR) + `" fill="none"/>` +
		face(fig.Face) +
		decoration +
		prop(fig.Prop)

	transform := fmt.Sprintf("translate(%s %s) scale(%s)", f(fig.X), f(fig.Y), f(fig.Scale))
	if math.Abs(fig.Tilt) > 0.01 {
		transform += fmt.Sprintf(" rotate(%s)", f(fig.Tilt))
	}

	return `<g transform="` + transform + `">` + body + `</g>`
}

// Render 把一組人畫成一張 SVG。用 currentColor 讓前端可以決定線條顏色。
func Render(scene Scene) string {
	var b strings.Builder

	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 480 270" width="480" height="270" role="img">`)
	b.WriteString(`<title>` + escape(scene.Caption) + `</title>`)
	b.WriteString(`<g fill="none" stroke="#e2e8f0" stroke-width="3.2" stroke-linecap="round" stroke-linejoin="round" color="#e2e8f0">`)

	for _, fig := range scene.Figures {
		b.WriteString(drawFigure(fig))
	}

	b.WriteString(`</g>`)

	if scene.Note != "" {
		b.WriteString(`<text x="240" y="256" font-size="15" text-anchor="middle" fill="#94a3b8" font-family="system-ui,sans-serif">` +
			escape(scene.Note) + `</text>`)
	}

	b.WriteString(`</svg>`)
	return b.String()
}

func escape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}
