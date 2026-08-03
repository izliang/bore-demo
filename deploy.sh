#!/bin/bash

# ===============================
# Force UTF-8 (encoding)
# ===============================
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8

# --- Configuration ---
PROJECT_NAME="bore-demo"
echo $PROJECT_NAME
UPLOAD_PATH="https://liang1-$PROJECT_NAME-main.app-base-platform.com/dev/deploy"
HEALTH_URL="https://liang1-$PROJECT_NAME-main.app-base-platform.com/ping"

PROJECT="cybertron"
RESTART="cd /app/ailab/cybertron && sh deploy_restart.sh"

#PATH=$(pwd)
#cd $PATH/$arg;
#echo $PATH/$arg

# --- Health check settings ---
MAX_RETRIES=60    # Max retry count
RETRY_DELAY=2     # Seconds between checks

# --- Timer init ---
START_TIME=$(date +%s.%N)
STEP_START_TIME=$(date +%s.%N)

MODE="git-diff"
if [ $# -gt 0 ]; then
    MODE="$1"
fi
TAR_FILE="deploy.tar.gz"
LIST_FILE="deploy_files.txt"

# ===============================
# 1. Pack
# ===============================
EXCLUDE_DIRS=(
   ".git"
  "__pycache__"
  "venv"
  ".venv"
  "logs"
  "deploy.sh"
  "deploy.tar.gz"
#  "./server"
)

# Build grep exclude pattern
EXCLUDE_GREP_PATTERN=$(printf "|^%s/" "${EXCLUDE_DIRS[@]}")
EXCLUDE_GREP_PATTERN=${EXCLUDE_GREP_PATTERN:1}


echo -e "\033[35m--- [Step 1] Packing ---\033[0m"
if [ "$MODE" = "git-diff" ]; then
    echo -e "\033[36mScanning for git changes...\033[0m"
    files=$(git ls-files --modified --others --exclude-standard \
      | grep -Ev "$EXCLUDE_GREP_PATTERN" \
      | grep -v "^$(basename "$TAR_FILE")$" \
      | grep -v "^$(basename "$LIST_FILE")$"
    )
    if [ -z "$files" ]; then
        echo -e "\033[32mNo changes detected. Exit.\033[0m"
        exit 0
    fi
    echo "$files" > "$LIST_FILE"
    tar -czf "$TAR_FILE" -T "$LIST_FILE"
else
    echo -e "\033[36mFull packing...\033[0m"
    TAR_EXCLUDES=()
    for dir in "${EXCLUDE_DIRS[@]}"; do
      TAR_EXCLUDES+=(--exclude="$dir")
    done
    tar -czf "$TAR_FILE" "${TAR_EXCLUDES[@]}" .
fi

STEP_END_TIME=$(date +%s.%N)
packTime=$(echo "$STEP_END_TIME - $STEP_START_TIME" | bc -l)
sizeMB=$(du -m "$TAR_FILE" | cut -f1)
printf "Done. Size: %.2f MB | Cost: %.2f s\n" "$sizeMB" "$packTime"

# ===============================
# 2. Upload & trigger
# ===============================
STEP_START_TIME=$(date +%s.%N)
echo -e "\n\033[35m--- [Step 2] Uploading & Triggering ---\033[0m"
# Submit request with curl
curl -k -s -X POST "$UPLOAD_PATH" \
    -F "file=@$TAR_FILE" \
    -F "project=$PROJECT" \
    -F "restart_cmd=$RESTART"

STEP_END_TIME=$(date +%s.%N)
uploadTime=$(echo "$STEP_END_TIME - $STEP_START_TIME" | bc -l)
printf "Done. Cost: %.2f s\n" "$uploadTime"

# Cleanup
[ -f "$TAR_FILE" ] && rm "$TAR_FILE"
[ -f "$LIST_FILE" ] && rm "$LIST_FILE"

# ===============================
# 3. Health check (compatible)
# ===============================
STEP_START_TIME=$(date +%s.%N)
echo -e "\n\033[35m--- [Step 3] Waiting for Service Health ---\033[0m"
success=false

echo -e "\033[90mMonitoring $HEALTH_URL\033[0m"

for ((i=1; i<=MAX_RETRIES; i++)); do
    # Health check via curl, 3s timeout
    response=$(curl -k -s -w "%{http_code}" -m 3 "$HEALTH_URL" 2>/dev/null)
    statusCode=${response: -3}

    if [ "$statusCode" = "200" ]; then
        echo -ne "\033[90m[$statusCode]\033[0m"
        success=true
        break
    elif [ -n "$statusCode" ] && [ "$statusCode" -ge 100 ] && [ "$statusCode" -lt 600 ]; then
        echo -ne "\033[90m[$statusCode]\033[0m"
    else
        # No valid status code; show error
        echo -ne "\033[33m[ERR: Connection failed]\033[0m"
    fi
    sleep "$RETRY_DELAY"
done

STEP_END_TIME=$(date +%s.%N)
healthTime=$(echo "$STEP_END_TIME - $STEP_START_TIME" | bc -l)

if [ "$success" = false ]; then
    printf "\n\n\033[31m[ERROR] Health check timed out after %.2f s.\033[0m\n" "$healthTime"
    exit 1
else
    echo -e "\n\n\033[32m[SUCCESS] Service is UP and responding 200 OK!\033[0m"
fi

# ===============================
# 4. Summary
# ===============================
END_TIME=$(date +%s.%N)
totalTime=$(echo "$END_TIME - $START_TIME" | bc -l)

echo -e "\n\033[90m$(printf '%.0s=' {1..30})\033[0m"
echo -e "\033[37m  DEPLOYMENT REPORT\033[0m"
echo -e "\033[90m$(printf '%.0s=' {1..30})\033[0m"
printf "1. Pack Cost      : %.2f s\n" "$packTime"
printf "2. Upload Cost    : %.2f s\n" "$uploadTime"
printf "3. Startup Cost   : %.2f s\n" "$healthTime"
echo -e "\033[90m$(printf '%.0s-' {1..30})\033[0m"
printf "Total Time        : \033[36m%.2f s\033[0m\n" "$totalTime"
echo -e "\033[90m$(printf '%.0s=' {1..30})\033[0m"