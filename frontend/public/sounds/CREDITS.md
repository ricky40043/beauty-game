# 音效來源與授權

## shutter.mp3 — 快門聲

| | |
|---|---|
| 原始檔案 | [File:Holga shuttersound.ogg](https://commons.wikimedia.org/wiki/File:Holga_shuttersound.ogg) |
| 來源 | Wikimedia Commons |
| 授權 | **Public domain（公有領域）** — Commons 中介資料標記 `Copyrighted: False` |
| 錄製者 | Doitashimashite（荷蘭文維基百科使用者） |
| 說明 | Holga 120S 相機的快門聲，錄於 2006-09-19。上傳者原話：<br>「I make this recording freely available to the internet community.」 |

公有領域不要求標示來源，這份紀錄純粹是為了日後能追溯素材出處。

### 做過的加工

原檔是 2.05 秒的 Ogg Vorbis，前後有環境音，直接播放會拖。處理步驟：

1. `silenceremove` 去掉前後靜音 → 0.55 秒
2. 尾端 55ms 淡出，避免結尾爆音
3. 音量 +1.1 倍（峰值 -0.9dB，留一點餘裕不削波）
4. 轉成 128kbps 單聲道 MP3 → 9.6KB

**為什麼轉 MP3**：Safari 對 Ogg Vorbis 的 `<audio>` 支援不可靠，而主畫面很可能就開在 Mac 上。MP3 所有瀏覽器都吃。
