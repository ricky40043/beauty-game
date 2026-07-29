package names

import (
	"fmt"
	"math/rand"
	"strings"
)

// 免取名模式用的暱稱素材：形容詞 + 動物，動物自帶 emoji 當頭像。
var adjectives = []string{
	"閃亮的", "害羞的", "優雅的", "淘氣的", "慵懶的", "神秘的", "澎湃的", "軟綿綿的",
	"發光的", "無敵的", "傳說中的", "偷偷摸摸的", "剛睡醒的", "興奮的", "冷靜的", "香噴噴的",
	"熱情的", "浮誇的", "低調的", "膨脹的", "唱歌的", "跳舞的", "會發電的", "拍照很兇的",
}

type animal struct {
	Name  string
	Emoji string
}

var animals = []animal{
	{"水獺", "🦦"}, {"柴犬", "🐕"}, {"貓咪", "🐱"}, {"熊貓", "🐼"}, {"企鵝", "🐧"},
	{"狐狸", "🦊"}, {"獅子", "🦁"}, {"兔子", "🐰"}, {"倉鼠", "🐹"}, {"樹懶", "🦥"},
	{"鯨魚", "🐳"}, {"章魚", "🐙"}, {"獨角獸", "🦄"}, {"恐龍", "🦖"}, {"無尾熊", "🐨"},
	{"刺蝟", "🦔"}, {"火鶴", "🦩"}, {"貓頭鷹", "🦉"}, {"羊駝", "🦙"}, {"海豚", "🐬"},
	{"青蛙", "🐸"}, {"小雞", "🐤"}, {"鴨子", "🦆"}, {"浣熊", "🦝"},
}

// Generate 產生一組不與 taken 重複的暱稱與頭像。
// taken 是房內既有暱稱（原樣，不含 emoji）。
func Generate(taken map[string]bool) (name string, avatar string) {
	for attempt := 0; attempt < 60; attempt++ {
		a := animals[rand.Intn(len(animals))]
		candidate := adjectives[rand.Intn(len(adjectives))] + a.Name
		if !taken[candidate] {
			return candidate, a.Emoji
		}
	}

	// 極端情況：素材撞光了就補流水號
	a := animals[rand.Intn(len(animals))]
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s%s%d", adjectives[rand.Intn(len(adjectives))], a.Name, i)
		if !taken[candidate] {
			return candidate, a.Emoji
		}
	}
}

// RandomAvatar 玩家自行取名時，仍配一個頭像 emoji
func RandomAvatar() string {
	return animals[rand.Intn(len(animals))].Emoji
}

// Sanitize 清理玩家自填的暱稱：去頭尾空白、砍換行、限制長度
func Sanitize(raw string) string {
	cleaned := strings.TrimSpace(strings.NewReplacer("\n", "", "\r", "", "\t", " ").Replace(raw))
	runes := []rune(cleaned)
	if len(runes) > 16 {
		cleaned = string(runes[:16])
	}
	return cleaned
}
