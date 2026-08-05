#!/usr/bin/env bash

set -euo pipefail

PROD=false
FORCE=false
SKIP_ENV_CHECK=false

for arg in "$@"; do
  case "$arg" in
    --prod)
      PROD=true
      ;;
    --force)
      FORCE=true
      ;;
    --skip-env-check)
      SKIP_ENV_CHECK=true
      ;;
    -h|--help)
      cat <<'EOF'
BaseGoApp 一键部署脚本 (macOS/Linux)

用法:
  ./deploy.sh [--prod] [--force] [--skip-env-check]

参数:
  --prod             使用生产环境配置 (docker-compose.prod.yml)
  --force            强制重新构建镜像（不使用缓存）
  --skip-env-check   跳过 .env 文件检查
EOF
      exit 0
      ;;
    *)
      echo "[ERROR] Unknown argument: $arg" >&2
      exit 1
      ;;
  esac
done

info() { echo "[INFO] $*"; }
ok() { echo "[OK] $*"; }
warn() { echo "[WARN] $*"; }
err() { echo "[ERROR] $*" >&2; }

generate_secret() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
  else
    LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 64
  fi
}

set_env_value() {
  local key="$1"
  local value="$2"
  local file="$3"
  if grep -q "^${key}=" "$file"; then
    sed -i.bak "s|^${key}=.*$|${key}=${value}|" "$file"
  else
    printf '%s=%s\n' "$key" "$value" >>"$file"
  fi
}

get_env_value() {
  local key="$1"
  local file="$2"
  awk -F= -v k="$key" '$1==k {sub(/^[^=]*=/, "", $0); print $0; exit}' "$file"
}

is_weak_jwt_secret() {
  local secret="$1"
  [[ -z "$secret" || "${#secret}" -lt 32 || "$secret" == "your-secret-key-change-in-production" ]]
}

is_weak_admin_password() {
  local username="$1"
  local password="$2"
  local username_lower
  local password_lower
  username_lower="$(printf '%s' "$username" | tr '[:upper:]' '[:lower:]')"
  password_lower="$(printf '%s' "$password" | tr '[:upper:]' '[:lower:]')"
  [[ -z "$password" || "${#password}" -lt 12 || ( "$username_lower" == "admin" && "$password_lower" == "admin" ) ]]
}

if command -v docker >/dev/null 2>&1; then
  DOCKER_CMD=(docker)
elif [[ -x /Applications/Docker.app/Contents/Resources/bin/docker ]]; then
  DOCKER_CMD=(/Applications/Docker.app/Contents/Resources/bin/docker)
else
  err "docker command not found. Please install Docker Desktop or Docker Engine."
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR"
cd "$PROJECT_ROOT"

if "${PROD}"; then
  COMPOSE_FILE="docker-compose.prod.yml"
  PROJECT_NAME="basegoapp-prod"
  info "使用生产环境配置: ${COMPOSE_FILE}"
else
  COMPOSE_FILE="docker-compose.yml"
  PROJECT_NAME="basegoapp"
  info "使用开发环境配置: ${COMPOSE_FILE}"
fi

if [[ ! -f "$COMPOSE_FILE" ]]; then
  err "找不到配置文件: ${COMPOSE_FILE}"
  exit 1
fi

compose() {
  "${DOCKER_CMD[@]}" compose "$@"
}

echo
echo "========================================"
echo "   BaseGoApp 一键部署脚本"
echo "========================================"
echo

info "步骤 1/4: 检查环境变量文件..."
if ! "${SKIP_ENV_CHECK}"; then
  if [[ ! -f ".env" ]]; then
    if [[ -f ".env.example" ]]; then
      warn ".env 文件不存在，正在从 .env.example 复制创建..."
      cp .env.example .env
      ok "已创建 .env 文件"

      jwt_secret="$(generate_secret)"
      admin_password="$(generate_secret | cut -c1-20)"
      set_env_value "DEFAULT_ADMIN_USERNAME" "admin" ".env"
      set_env_value "JWT_SECRET" "$jwt_secret" ".env"
      set_env_value "DEFAULT_ADMIN_PASSWORD" "$admin_password" ".env"
      rm -f .env.bak

      ok "已自动生成强随机 JWT_SECRET"
      ok "已自动生成初始管理员密码 DEFAULT_ADMIN_PASSWORD"
      warn "请妥善保存并按需修改 .env 中的安全配置"

      read -r -p "是否继续部署？(y/N) " answer
      if [[ "$answer" != "y" && "$answer" != "Y" ]]; then
        info "已取消部署，请修改 .env 文件后重新运行脚本"
        exit 0
      fi
    else
      err ".env 和 .env.example 文件都不存在！"
      exit 1
    fi
  else
    ok ".env 文件已存在"
    current_jwt_secret="$(get_env_value "JWT_SECRET" ".env")"
    if is_weak_jwt_secret "$current_jwt_secret"; then
      warn "检测到 JWT_SECRET 过弱或仍为默认值，正在自动修复..."
      set_env_value "JWT_SECRET" "$(generate_secret)" ".env"
      rm -f .env.bak
      ok "已自动更新 JWT_SECRET（满足 release 模式要求）"
    fi

    current_admin_username="$(get_env_value "DEFAULT_ADMIN_USERNAME" ".env")"
    current_admin_password="$(get_env_value "DEFAULT_ADMIN_PASSWORD" ".env")"

    if [[ -z "$current_admin_username" ]]; then
      warn "检测到 DEFAULT_ADMIN_USERNAME 为空，正在自动设置为 admin..."
      set_env_value "DEFAULT_ADMIN_USERNAME" "admin" ".env"
      rm -f .env.bak
      ok "已自动更新 DEFAULT_ADMIN_USERNAME=admin"
      current_admin_username="admin"
    fi

    if is_weak_admin_password "$current_admin_username" "$current_admin_password"; then
      warn "检测到 DEFAULT_ADMIN_PASSWORD 过弱，正在自动修复..."
      set_env_value "DEFAULT_ADMIN_PASSWORD" "$(generate_secret | cut -c1-20)" ".env"
      rm -f .env.bak
      ok "已自动更新 DEFAULT_ADMIN_PASSWORD（满足安全要求）"
    fi
  fi
else
  warn "已跳过环境变量文件检查"
fi

echo
info "步骤 2/4: 停止并移除旧容器..."
if compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down --remove-orphans; then
  ok "已停止并移除旧容器"
else
  warn "没有找到运行中的容器或移除失败（可忽略）"
fi

echo
info "步骤 3/4: 移除旧镜像..."
image_ids="$("${DOCKER_CMD[@]}" images --filter "reference=${PROJECT_NAME}*" -q || true)"
if [[ -n "$image_ids" ]]; then
  "${DOCKER_CMD[@]}" rmi $image_ids -f || true
  ok "已移除旧镜像"
else
  info "没有找到需要移除的旧镜像"
fi

dangling_ids="$("${DOCKER_CMD[@]}" images -f "dangling=true" -q || true)"
if [[ -n "$dangling_ids" ]]; then
  "${DOCKER_CMD[@]}" rmi $dangling_ids -f || true
  ok "已清理悬空镜像"
fi

echo
info "步骤 4/4: 构建并启动新容器..."
if "${FORCE}"; then
  info "使用 --no-cache 强制重新构建..."
  compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" build --no-cache
fi

compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" up -d --build
ok "容器启动成功！"

echo
echo "========================================"
echo "   部署完成！"
echo "========================================"
echo

info "容器状态:"
compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps

echo
ok "应用访问地址:"
echo "  - 应用: http://localhost"
echo "  - API: http://localhost/api"
echo "  - Swagger: http://localhost/swagger/index.html"
echo
info "查看日志: docker compose -f ${COMPOSE_FILE} -p ${PROJECT_NAME} logs -f"
info "停止服务: docker compose -f ${COMPOSE_FILE} -p ${PROJECT_NAME} down"
echo
