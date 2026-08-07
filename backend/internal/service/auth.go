package service

import (
	"errors"
	"strings"
	"time"

	"basegoapp/config"
	"basegoapp/internal/model"
	"basegoapp/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService 认证服务
type AuthService struct {
	userRepo  *repository.UserRepository
	jwtSecret string
}

const bootstrapAdminUsername = "admin"
const bootstrapAdminPassword = "admin"

// NewAuthService 创建认证服务实例
func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{
		userRepo:  repository.NewUserRepository(),
		jwtSecret: cfg.JWTSecret,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// SetupAdminRequest 首次登录后设置正式管理员凭据。
type SetupAdminRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
}

// TokenResponse Token 响应
type TokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// UserResponse 用户响应
type UserResponse struct {
	ID                 uint   `json:"id"`
	Username           string `json:"username"`
	IsAdmin            bool   `json:"is_admin"`
	RequiresAdminSetup bool   `json:"requires_admin_setup"`
	CreatedAt          string `json:"created_at"`
}

// Register 用户注册
func (s *AuthService) Register(req *RegisterRequest) (*UserResponse, error) {
	// 检查用户名是否已存在
	exists, err := s.userRepo.ExistsByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("用户名已存在")
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username: req.Username,
		Password: string(hashedPassword),
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return userResponse(user), nil
}

// Login 用户登录
func (s *AuthService) Login(req *LoginRequest) (*TokenResponse, error) {
	user, err := s.userRepo.FindByUsername(req.Username)
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	return s.issueToken(user)
}

func (s *AuthService) issueToken(user *model.User) (*TokenResponse, error) {
	expiresAt := time.Now().Add(24 * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":       user.ID,
		"username":      user.Username,
		"is_admin":      user.IsAdmin,
		"token_version": user.TokenVersion,
		"exp":           expiresAt.Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

// GetProfile 获取用户信息
func (s *AuthService) GetProfile(userID uint) (*UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("用户不存在")
	}

	return userResponse(user), nil
}

// SetupAdmin 将一次性引导管理员更新为正式管理员，并签发新 Token。
func (s *AuthService) SetupAdmin(userID uint, req *SetupAdminRequest) (*TokenResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("用户不存在")
	}
	if !user.IsAdmin || !user.RequiresAdminSetup {
		return nil, errors.New("管理员初始化已完成")
	}

	username := strings.TrimSpace(req.Username)
	if len(username) < 3 || len(username) > 50 {
		return nil, errors.New("管理员用户名长度必须在 3 到 50 位之间")
	}
	if len(req.Password) < 6 {
		return nil, errors.New("管理员密码长度至少 6 位")
	}
	if username != user.Username {
		exists, existsErr := s.userRepo.ExistsByUsername(username)
		if existsErr != nil {
			return nil, existsErr
		}
		if exists {
			return nil, errors.New("用户名已存在")
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user.Username = username
	user.Password = string(hashedPassword)
	user.RequiresAdminSetup = false
	user.TokenVersion++
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}
	return s.issueToken(user)
}

// ChangePassword 修改密码
func (s *AuthService) ChangePassword(userID uint, req *ChangePasswordRequest) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("用户不存在")
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return errors.New("原密码错误")
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	user.TokenVersion++
	return s.userRepo.Update(user)
}

// Logout 递增认证版本，使当前 Bearer Token 及同版本 Token 立即失效。
func (s *AuthService) Logout(userID uint) error {
	return s.userRepo.IncrementTokenVersion(userID)
}

// InitBootstrapAdmin 在空数据库中创建一次性引导管理员 admin/admin。
func (s *AuthService) InitBootstrapAdmin() error {
	count, err := s.userRepo.CountAdmins()
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(bootstrapAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user := &model.User{
		Username:           bootstrapAdminUsername,
		Password:           string(hashedPassword),
		IsAdmin:            true,
		RequiresAdminSetup: true,
	}
	return s.userRepo.Create(user)
}

// ListUsers 获取用户列表
func (s *AuthService) ListUsers() ([]UserResponse, error) {
	users, err := s.userRepo.FindAll()
	if err != nil {
		return nil, err
	}

	var result []UserResponse
	for _, user := range users {
		result = append(result, *userResponse(&user))
	}
	return result, nil
}

func userResponse(user *model.User) *UserResponse {
	return &UserResponse{
		ID:                 user.ID,
		Username:           user.Username,
		IsAdmin:            user.IsAdmin,
		RequiresAdminSetup: user.RequiresAdminSetup,
		CreatedAt:          user.CreatedAt.Format(time.RFC3339),
	}
}

// SetUserRole 设置用户角色
func (s *AuthService) SetUserRole(userID uint, isAdmin bool) error {
	return s.userRepo.SetAdmin(userID, isAdmin)
}
