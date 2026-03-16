#!/bin/bash
# =====================================================
# 部署 Cocos Build 到 Server
# 在 Windows 上使用 Git Bash / WSL 執行
#
# 用法: ./deploy.sh
# =====================================================

SERVER="chris@192.168.50.202"
REMOTE_DIR="/home/chris/go/game/webMajiang/build/web-mobile"
LOCAL_DIR="$(dirname "$0")/build/web-mobile"

if [ ! -d "$LOCAL_DIR" ]; then
    echo "❌ 找不到 build 目錄: $LOCAL_DIR"
    echo "   請先在 Cocos Creator 中 Build (web-mobile)"
    exit 1
fi

echo "📦 同步 Build 到 Server..."
echo "   本地: $LOCAL_DIR"
echo "   遠端: $SERVER:$REMOTE_DIR"

# rsync: 同步所有檔案，刪除遠端多餘的舊檔案
rsync -avz --delete \
    "$LOCAL_DIR/" \
    "$SERVER:$REMOTE_DIR/"

echo ""
echo "✅ 部署完成！"
echo "   URL: https://lobby.cxwoo.com/game/"
echo ""
echo "💡 提醒: 瀏覽器請按 Ctrl+Shift+R 強制刷新以清除快取"
