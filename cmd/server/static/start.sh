#!/bin/bash

# zebra-ops Frontend 启动脚本

echo "▸ Starting zebra-ops Frontend..."
#echo "[目录] 当前目录: $(pwd)"
echo "◉ Frontend will start at http://localhost:8080"
echo "→ Make sure backend is running at http://localhost:6872"
echo ""

# 检查 Python 是否可用
if command -v python3 &> /dev/null; then
    echo "✓ Starting with Python3..."
    python3 -m http.server 8080
elif command -v python &> /dev/null; then
    echo "✓ Starting with Python..."
    python -m http.server 8080
elif command -v node &> /dev/null; then
    echo "✓ Starting with Node.js..."
    npx http-server -p 8080
else
    echo "✗ Error: Python or Node.js not found"
    echo "Please install Python3 or Node.js to run this project"
    exit 1
fi
