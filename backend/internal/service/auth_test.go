package service

import (
	"testing"

	"basegoapp/config"
	"basegoapp/internal/model"
	"basegoapp/pkg/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuthTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:auth-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	database.DB = db
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&model.User{}).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.DB = nil })
}

func TestBootstrapAdminMustSetFormalCredentials(t *testing.T) {
	setupAuthTestDB(t)
	authService := NewAuthService(&config.Config{JWTSecret: "test-jwt-secret"})
	if err := authService.InitBootstrapAdmin(); err != nil {
		t.Fatal(err)
	}

	if _, err := authService.Login(&LoginRequest{Username: "admin", Password: "admin"}); err != nil {
		t.Fatalf("bootstrap login failed: %v", err)
	}
	bootstrapUser, err := authService.userRepo.FindByUsername("admin")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := authService.GetProfile(bootstrapUser.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !profile.IsAdmin || !profile.RequiresAdminSetup {
		t.Fatal("bootstrap account was not marked for administrator setup")
	}

	if _, err := authService.SetupAdmin(bootstrapUser.ID, &SetupAdminRequest{Username: "owner", Password: "StrongPassword123"}); err != nil {
		t.Fatal(err)
	}
	if _, err := authService.Login(&LoginRequest{Username: "admin", Password: "admin"}); err == nil {
		t.Fatal("bootstrap credentials still worked after setup")
	}
	if _, err := authService.Login(&LoginRequest{Username: "owner", Password: "StrongPassword123"}); err != nil {
		t.Fatalf("formal administrator login failed: %v", err)
	}
	formalUser, err := authService.userRepo.FindByUsername("owner")
	if err != nil {
		t.Fatal(err)
	}
	if formalUser.RequiresAdminSetup {
		t.Fatal("administrator setup flag was not cleared")
	}
	versionBeforeLogout := formalUser.TokenVersion
	if err := authService.Logout(formalUser.ID); err != nil {
		t.Fatal(err)
	}
	loggedOutUser, err := authService.userRepo.FindByID(formalUser.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loggedOutUser.TokenVersion != versionBeforeLogout+1 {
		t.Fatal("logout did not invalidate issued tokens")
	}
	if _, err := authService.SetupAdmin(formalUser.ID, &SetupAdminRequest{Username: "other", Password: "AnotherPassword123"}); err == nil {
		t.Fatal("administrator setup was allowed more than once")
	}
}

func TestBootstrapAdminIsCreatedWhenUsersExistWithoutAdministrator(t *testing.T) {
	setupAuthTestDB(t)
	authService := NewAuthService(&config.Config{JWTSecret: "test-jwt-secret"})
	if _, err := authService.Register(&RegisterRequest{Username: "member", Password: "member-password"}); err != nil {
		t.Fatal(err)
	}
	if err := authService.InitBootstrapAdmin(); err != nil {
		t.Fatal(err)
	}
	bootstrapUser, err := authService.userRepo.FindByUsername("admin")
	if err != nil {
		t.Fatal(err)
	}
	if !bootstrapUser.IsAdmin || !bootstrapUser.RequiresAdminSetup {
		t.Fatal("bootstrap administrator was not created")
	}
}
