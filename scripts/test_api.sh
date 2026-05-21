#!/bin/bash
set -euo pipefail

BASE="${1:-http://localhost}"
API="$BASE/api/v1"

GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; NC='\033[0m'
PASS=0; FAIL=0

jget() {
  local json="$1" key="$2"
  printf '%s' "$json" \
    | tr -d '\n\r' \
    | sed -n 's/.*"'"$key"'":"\([^"]*\)".*/\1/p' \
    | head -1
}

check() {
  local name="$1" expected="$2" actual="$3"
  if echo "$actual" | grep -qE "$expected"; then
    echo -e "${GREEN}✅ PASS${NC} — $name"; PASS=$((PASS+1))
  else
    echo -e "${RED}❌ FAIL${NC} — $name"
    echo -e "   ${YELLOW}Ожидалось:${NC} $expected"
    echo -e "   ${YELLOW}Получено: ${NC} $(echo "$actual" | head -c 300)"
    FAIL=$((FAIL+1))
  fi
}

post_json() {
  local url="$1" data="$2"; shift 2
  curl -s -X POST "$url" -H "Content-Type: application/json" --data-raw "$data" "$@"
}
get()      { local url="$1"; shift; curl -s "$url" "$@"; }
get_code() { local url="$1"; shift; curl -s -o /dev/null -w "%{http_code}" "$url" "$@"; }
section()  { echo ""; echo -e "${BLUE}━━━ $1 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; }

section "1. HEALTH CHECK"
R=$(get "$BASE/health")
check "GET /health → status ok" "ok" "$R"

section "2. РЕГИСТРАЦИЯ"
# $RANDOM$RANDOM — уникальный ID без научной нотации (в отличие от date +%s)
RUN_ID="${RANDOM}${RANDOM}"
ALICE_USER="alice_${RUN_ID}"; BOB_USER="bob_${RUN_ID}"
ALICE_FP="fp-alice-${RUN_ID}"; BOB_FP="fp-bob-${RUN_ID}"

ALICE_REG=$(post_json "$API/auth/register" \
  '{"username":"'"$ALICE_USER"'","password":"securepass123","device_name":"Test Device","device_fp":"'"$ALICE_FP"'","platform":"android"}')
echo -e "  ${CYAN}alice reg:${NC} $(echo "$ALICE_REG" | head -c 120)..."
check "POST /auth/register alice → access_token" "access_token" "$ALICE_REG"
check "POST /auth/register alice → username" "$ALICE_USER" "$ALICE_REG"
ALICE_ACCESS=$(jget "$ALICE_REG" "access_token")
ALICE_REFRESH=$(jget "$ALICE_REG" "refresh_token")
ALICE_ID=$(jget "$ALICE_REG" "id")
echo -e "  ${YELLOW}alice_id:${NC}    $ALICE_ID"
echo -e "  ${YELLOW}access len:${NC}  ${#ALICE_ACCESS}"
echo -e "  ${YELLOW}refresh len:${NC} ${#ALICE_REFRESH}"

BOB_REG=$(post_json "$API/auth/register" \
  '{"username":"'"$BOB_USER"'","password":"securepass456","device_name":"Bob Phone","device_fp":"'"$BOB_FP"'","platform":"ios"}')
check "POST /auth/register bob → access_token" "access_token" "$BOB_REG"
BOB_ACCESS=$(jget "$BOB_REG" "access_token")
BOB_ID=$(jget "$BOB_REG" "id")
echo -e "  ${YELLOW}bob_id:${NC}      $BOB_ID"

DUP=$(post_json "$API/auth/register" \
  '{"username":"'"$ALICE_USER"'","password":"anotherpass123","device_name":"Other","device_fp":"fp-other-999","platform":"android"}')
check "POST /auth/register дубликат → already taken" "already taken" "$DUP"

section "3. ЛОГИН"
LOGIN_OK=$(post_json "$API/auth/login" \
  '{"username":"'"$ALICE_USER"'","password":"securepass123","device_fp":"'"$ALICE_FP"'"}')
check "POST /auth/login → access_token" "access_token" "$LOGIN_OK"
ALICE_ACCESS=$(jget "$LOGIN_OK" "access_token")
ALICE_REFRESH=$(jget "$LOGIN_OK" "refresh_token")
echo -e "  ${YELLOW}refresh len after login:${NC} ${#ALICE_REFRESH}"

LOGIN_BAD=$(post_json "$API/auth/login" \
  '{"username":"'"$ALICE_USER"'","password":"wrongpassword","device_fp":"'"$ALICE_FP"'"}')
check "POST /auth/login неверный пароль → invalid credentials" "invalid credentials" "$LOGIN_BAD"

section "4. ЗАЩИЩЁННЫЕ МАРШРУТЫ (Auth Guard)"
ME=$(get "$API/auth/me" -H "Authorization: Bearer $ALICE_ACCESS")
check "GET /auth/me с токеном → user_id" "user_id" "$ME"
check "GET /auth/me без токена → 401" "401" "$(get_code "$API/auth/me")"
check "GET /auth/me с невалидным токеном → 401" "401" \
  "$(get_code "$API/auth/me" -H "Authorization: Bearer invalid.token.here")"

section "5. REFRESH ТОКЕН"
echo -e "  ${CYAN}Отправляем refresh длиной ${#ALICE_REFRESH} символов${NC}"
REFRESH_R=$(post_json "$API/auth/refresh" '{"refresh_token":"'"$ALICE_REFRESH"'"}')
check "POST /auth/refresh → новый access_token" "access_token" "$REFRESH_R"
if echo "$REFRESH_R" | grep -q "access_token"; then
  _A=$(jget "$REFRESH_R" "access_token"); _R=$(jget "$REFRESH_R" "refresh_token")
  [ -n "$_A" ] && ALICE_ACCESS="$_A"
  [ -n "$_R" ] && ALICE_REFRESH="$_R"
  echo -e "  ${GREEN}Токены обновлены${NC}"
else
  echo -e "  ${YELLOW}Refresh не удался — используем токен из login${NC}"
fi

section "6. ПОИСК ПОЛЬЗОВАТЕЛЕЙ"
check "GET /users/search?q=bob → bob найден" "$BOB_USER" \
  "$(get "$API/users/search?q=bob" -H "Authorization: Bearer $ALICE_ACCESS")"
check "GET /users/search?q=alice → alice найдена" "$ALICE_USER" \
  "$(get "$API/users/search?q=alice" -H "Authorization: Bearer $ALICE_ACCESS")"
check "GET /users/search?q=x (1 символ) → 400" "400" \
  "$(get_code "$API/users/search?q=x" -H "Authorization: Bearer $ALICE_ACCESS")"

section "7. E2EE КЛЮЧИ"
KEYS_UP=$(post_json "$API/users/keys" \
  '{"identity_key":"alice_identity_key_b64==","signed_prekey":"alice_signed_prekey_b64==","prekey_sig":"alice_prekey_sig_b64==","one_time_keys":["otk1aa==","otk2aa==","otk3aa=="]}' \
  -H "Authorization: Bearer $ALICE_ACCESS")
check "POST /users/keys → ok" "ok" "$KEYS_UP"
check "GET /users/:id/keys → identity_key" "identity_key" \
  "$(get "$API/users/$ALICE_ID/keys" -H "Authorization: Bearer $BOB_ACCESS")"

section "8. ЧАТЫ"
CHAT_CREATE=$(post_json "$API/chats" \
  '{"type":"direct","member_ids":["'"$BOB_ID"'"]}' \
  -H "Authorization: Bearer $ALICE_ACCESS")
check "POST /chats direct → создан" "direct" "$CHAT_CREATE"
CHAT_ID=$(jget "$CHAT_CREATE" "id")
echo -e "  ${YELLOW}chat_id:${NC} $CHAT_ID"
CHAT_DUP=$(post_json "$API/chats" \
  '{"type":"direct","member_ids":["'"$BOB_ID"'"]}' \
  -H "Authorization: Bearer $ALICE_ACCESS")
check "POST /chats повторно → тот же chat_id" "$CHAT_ID" "$(jget "$CHAT_DUP" "id")"
check "GET /chats → chats массив" "chats" \
  "$(get "$API/chats" -H "Authorization: Bearer $ALICE_ACCESS")"

section "9. СООБЩЕНИЯ"
check "GET /chats/:id/messages → messages массив" "messages" \
  "$(get "$API/chats/$CHAT_ID/messages" -H "Authorization: Bearer $ALICE_ACCESS")"
check "POST /chats/:id/read → ok" "ok" \
  "$(post_json "$API/chats/$CHAT_ID/read" '{}' -H "Authorization: Bearer $ALICE_ACCESS")"
check "GET /chats/несуществующий → 403" "403" \
  "$(get_code "$API/chats/00000000-0000-0000-0000-000000000000/messages" -H "Authorization: Bearer $ALICE_ACCESS")"

section "10. LOGOUT"
LOGIN2=$(post_json "$API/auth/login" \
  '{"username":"'"$ALICE_USER"'","password":"securepass123","device_fp":"'"$ALICE_FP"'"}')
FRESH_REFRESH=$(jget "$LOGIN2" "refresh_token")
check "POST /auth/logout → logged out" "logged out" \
  "$(post_json "$API/auth/logout" '{"refresh_token":"'"$FRESH_REFRESH"'"}')"
check "POST /auth/refresh после logout → ошибка" "not found|invalid|expired|session" \
  "$(post_json "$API/auth/refresh" '{"refresh_token":"'"$FRESH_REFRESH"'"}')"

section "ИТОГО"
echo ""
echo -e "  Всего тестов: $((PASS+FAIL))"
echo -e "  ${GREEN}Пройдено:  $PASS${NC}"
if [ "$FAIL" -gt 0 ]; then
  echo -e "  ${RED}Провалено: $FAIL${NC}"; exit 1
else
  echo -e "  ${GREEN}Провалено: 0 — всё работает! 🎉${NC}"
fi