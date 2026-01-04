# BaseGoApp 一键部署脚本
# 用法: .\deploy.ps1 [-Prod] [-Force] [-SkipEnvCheck]
# 参数:
#   -Prod          使用生产环境配置 (docker-compose.prod.yml)
#   -Force         强制重新构建镜像（不使用缓存）
#   -SkipEnvCheck  跳过环境变量文件检查

param(
    [switch]$Prod,
    [switch]$Force,
    [switch]$SkipEnvCheck
)

# 设置脚本在遇到错误时停止执行
$ErrorActionPreference = "Stop"

# 颜色输出函数
function Write-Info { param($msg) Write-Host "[INFO] $msg" -ForegroundColor Cyan }
function Write-Success { param($msg) Write-Host "[OK] $msg" -ForegroundColor Green }
function Write-Warning { param($msg) Write-Host "[WARN] $msg" -ForegroundColor Yellow }
function Write-Error { param($msg) Write-Host "[ERROR] $msg" -ForegroundColor Red }

# 脚本所在目录（项目根目录）
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ScriptDir

Write-Host ""
Write-Host "========================================" -ForegroundColor Magenta
Write-Host "   BaseGoApp 一键部署脚本" -ForegroundColor Magenta
Write-Host "========================================" -ForegroundColor Magenta
Write-Host ""

# 选择 docker-compose 配置文件
if ($Prod) {
    $ComposeFile = "docker-compose.prod.yml"
    $ProjectName = "basegoapp-prod"
    Write-Info "使用生产环境配置: $ComposeFile"
} else {
    $ComposeFile = "docker-compose.yml"
    $ProjectName = "basegoapp"
    Write-Info "使用开发环境配置: $ComposeFile"
}

# 检查 docker-compose 文件是否存在
if (-not (Test-Path $ComposeFile)) {
    Write-Error "找不到配置文件: $ComposeFile"
    exit 1
}

# ========================================
# 步骤 1: 检查并创建 .env 文件
# ========================================
Write-Host ""
Write-Info "步骤 1/4: 检查环境变量文件..."

$EnvFile = ".env"
$EnvExampleFile = ".env.example"

if (-not $SkipEnvCheck) {
    if (-not (Test-Path $EnvFile)) {
        if (Test-Path $EnvExampleFile) {
            Write-Warning ".env 文件不存在，正在从 .env.example 复制创建..."
            Copy-Item $EnvExampleFile $EnvFile
            Write-Success "已创建 .env 文件"
            Write-Warning "请检查并修改 .env 文件中的配置（特别是敏感信息如 JWT_SECRET）"
            
            # 询问用户是否继续
            $continue = Read-Host "是否继续部署？(y/N)"
            if ($continue -ne "y" -and $continue -ne "Y") {
                Write-Info "已取消部署，请修改 .env 文件后重新运行脚本"
                exit 0
            }
        } else {
            Write-Error ".env 和 .env.example 文件都不存在！"
            exit 1
        }
    } else {
        Write-Success ".env 文件已存在"
    }
} else {
    Write-Warning "已跳过环境变量文件检查"
}

# ========================================
# 步骤 2: 停止并移除旧容器
# ========================================
Write-Host ""
Write-Info "步骤 2/4: 停止并移除旧容器..."

try {
    # 使用 docker-compose down 停止并移除容器、网络
    docker-compose -f $ComposeFile -p $ProjectName down --remove-orphans 2>$null
    Write-Success "已停止并移除旧容器"
} catch {
    Write-Warning "没有找到运行中的容器或移除失败（可忽略）"
}

# ========================================
# 步骤 3: 移除旧镜像
# ========================================
Write-Host ""
Write-Info "步骤 3/4: 移除旧镜像..."

try {
    # 获取项目相关的镜像
    $images = docker images --filter "reference=${ProjectName}*" -q 2>$null
    if ($images) {
        docker rmi $images -f 2>$null
        Write-Success "已移除旧镜像"
    } else {
        Write-Info "没有找到需要移除的旧镜像"
    }
    
    # 清理悬空镜像（可选）
    $danglingImages = docker images -f "dangling=true" -q 2>$null
    if ($danglingImages) {
        docker rmi $danglingImages -f 2>$null
        Write-Success "已清理悬空镜像"
    }
} catch {
    Write-Warning "移除镜像时出现警告（可忽略）"
}

# ========================================
# 步骤 4: 构建并启动新容器
# ========================================
Write-Host ""
Write-Info "步骤 4/4: 构建并启动新容器..."

# 构建参数
$buildArgs = @("-f", $ComposeFile, "-p", $ProjectName)

if ($Force) {
    Write-Info "使用 --no-cache 强制重新构建..."
    $buildArgs += @("build", "--no-cache")
    & docker-compose @buildArgs
    
    if ($LASTEXITCODE -ne 0) {
        Write-Error "构建失败！"
        exit 1
    }
}

# 启动容器
$upArgs = @("-f", $ComposeFile, "-p", $ProjectName, "up", "-d", "--build")
& docker-compose @upArgs

if ($LASTEXITCODE -ne 0) {
    Write-Error "启动容器失败！"
    exit 1
}

Write-Success "容器启动成功！"

# ========================================
# 显示运行状态
# ========================================
Write-Host ""
Write-Host "========================================" -ForegroundColor Magenta
Write-Host "   部署完成！" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Magenta
Write-Host ""

Write-Info "容器状态:"
docker-compose -f $ComposeFile -p $ProjectName ps

Write-Host ""
if ($Prod) {
    Write-Success "应用已在生产模式下启动"
    Write-Host "  - 应用: http://localhost" -ForegroundColor Cyan
    Write-Host "  - API: http://localhost/api" -ForegroundColor Cyan
    Write-Host "  - Swagger: http://localhost/swagger/index.html" -ForegroundColor Cyan
} else {
    Write-Success "应用已在开发模式下启动"
    Write-Host "  - 应用: http://localhost" -ForegroundColor Cyan
    Write-Host "  - API: http://localhost/api" -ForegroundColor Cyan
    Write-Host "  - Swagger: http://localhost/swagger/index.html" -ForegroundColor Cyan
    Write-Host "  - 数据库: localhost:5432" -ForegroundColor Cyan
}

Write-Host ""
Write-Info "查看日志: docker-compose -f $ComposeFile -p $ProjectName logs -f"
Write-Info "停止服务: docker-compose -f $ComposeFile -p $ProjectName down"
Write-Host ""
