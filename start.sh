#!/usr/bin/env bash
# 本機開發：同時啟動 Go 後端（:8081）與 Vite 前端（:3344）
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🧹 清掉舊的行程…"
lsof -ti:8081 | xargs kill -9 2>/dev/null || true
lsof -ti:3344 | xargs kill -9 2>/dev/null || true

cleanup() {
  echo ""
  echo "🛑 關閉中…"
  kill "${BACKEND_PID:-}" "${FRONTEND_PID:-}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "🚀 啟動後端 :8081"
(cd "$ROOT/backend" && go run cmd/main.go) &
BACKEND_PID=$!

sleep 2

echo "🎨 啟動前端 :3344"
(cd "$ROOT/frontend" && npm run dev) &
FRONTEND_PID=$!

echo ""
echo "─────────────────────────────────────────"
echo "  主畫面：http://localhost:3344"
echo "  手機測試：用同網段 IP 連 http://<你的IP>:3344"
echo "  注意：手機走 http 時網頁相機會被瀏覽器擋，"
echo "        會自動退回「開啟系統相機」，功能一樣完整。"
echo "─────────────────────────────────────────"
echo ""

wait
