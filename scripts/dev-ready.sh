#!/usr/bin/env bash
set -euo pipefail

FRONTEND_URL="http://localhost:5173"
BACKEND_URL="http://localhost:8080"

# Poll until both Vite frontend and Go backend respond (up to 30s)
frontend_ready=0
backend_ready=0

for i in $(seq 1 60); do
  if [ "$frontend_ready" -eq 0 ] && curl -s -o /dev/null -m 1 "$FRONTEND_URL" 2>/dev/null; then
    frontend_ready=1
  fi
  if [ "$backend_ready" -eq 0 ] && curl -s -o /dev/null -m 1 "$BACKEND_URL" 2>/dev/null; then
    backend_ready=1
  fi

  if [ "$frontend_ready" -eq 1 ] && [ "$backend_ready" -eq 1 ]; then
    break
  fi
  sleep 0.5
done

# Short sleep so trailing Air/Go startup log lines finish outputting
sleep 0.4

BOLD=$'\033[1m'
GREEN=$'\033[1;32m'
CYAN=$'\033[1;36m'
YELLOW=$'\033[1;33m'
DIM=$'\033[2m'
RESET=$'\033[0m'

printf "\n"
printf "%b┌────────────────────────────────────────────────────────┐%b\n" "$GREEN" "$RESET"
printf "%b│%b                                                        %b│%b\n" "$GREEN" "$RESET" "$GREEN" "$RESET"
printf "%b│%b   %b🚀 Gmail DeepClean Dev Server Ready%b                  %b│%b\n" "$GREEN" "$RESET" "$BOLD" "$RESET" "$GREEN" "$RESET"
printf "%b│%b                                                        %b│%b\n" "$GREEN" "$RESET" "$GREEN" "$RESET"
printf "%b│%b   %b👉 Open this URL in your browser:%b                    %b│%b\n" "$GREEN" "$RESET" "$YELLOW" "$RESET" "$GREEN" "$RESET"
printf "%b│%b      %b%s/%b                            %b│%b\n" "$GREEN" "$RESET" "$CYAN$BOLD" "$FRONTEND_URL" "$RESET" "$GREEN" "$RESET"
printf "%b│%b                                                        %b│%b\n" "$GREEN" "$RESET" "$GREEN" "$RESET"
printf "%b│%b   %b• Frontend (Vue + HMR):%b  %s/      %b│%b\n" "$GREEN" "$RESET" "$DIM" "$RESET" "$FRONTEND_URL" "$GREEN" "$RESET"
printf "%b│%b   %b• Backend API (Go):%b      %s/      %b│%b\n" "$GREEN" "$RESET" "$DIM" "$RESET" "$BACKEND_URL" "$GREEN" "$RESET"
printf "%b│%b                                                        %b│%b\n" "$GREEN" "$RESET" "$GREEN" "$RESET"
printf "%b└────────────────────────────────────────────────────────┘%b\n" "$GREEN" "$RESET"
printf "\n"
