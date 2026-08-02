#!/bin/bash


BASE_DIR=/app/ailab/cybertron
cd "$BASE_DIR"

mkdir -p "$BASE_DIR/logs/cybertron"

APP_CMD="go run cmd/demo/main.go"
LOG_FILE="$BASE_DIR/logs/cybertron/stdout.log"

echo "Stopping old go processes..."

# 杀死旧进程
pkill -TERM -f "main" || true
sleep 2

pkill -KILL -f "main" || true

setsid bash -c "
  exec $APP_CMD >> '$LOG_FILE' 2>&1
" &
echo "Starting new process... $APP_CMD >> '$LOG_FILE' 2>&1 "
