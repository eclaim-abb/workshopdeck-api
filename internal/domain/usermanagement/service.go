package usermanagement

import (
	"eclaim-workshop-deck-api/internal/domain/auth"
	"eclaim-workshop-deck-api/internal/domain/email"
	"eclaim-workshop-deck-api/internal/models"
	"eclaim-workshop-deck-api/pkg/utils"
	"errors"
	"fmt"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) AddUser(id uint, req AddUserRequest) (*models.User, error) {
	if req.UserName == "" {
		return nil, errors.New("User name is required")
	}
	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	if req.RoleNo == 0 {
		return nil, errors.New("role no is required")
	}
	if req.Email == "" {
		return nil, errors.New("Email is required")
	}
	if req.CreatedBy == 0 {
		return nil, errors.New("Created by is required")
	}

	str, pwd, err := utils.GenerateRandomPassword(32)
	if err != nil {
		return nil, errors.New("failed to generate password")
	}

	user := &models.User{
		UserProfileNo: &req.UserProfileNo,
		RoleNo:        req.RoleNo,
		UserName:      req.Name,
		Email:         req.Email,
		UserId:        req.UserName,
		CreatedBy:     &req.CreatedBy,
		Password:      pwd,
	}
	authRepo := auth.NewRepository(s.repo.db, nil)
	err = authRepo.Create(user)
	if err != nil {
		return nil, err
	}
	emailService := email.NewEmailService()
	if err := emailService.SendCreatedUser(req.Email, req.Name, req.Email, str); err != nil {
		return user, err
	}

	return user, nil
}

func (s *Service) GetWorkshopUsers(workshopId uint) ([]models.User, error) {
	return s.repo.GetWorkshopUsers(workshopId)
}

func (s *Service) GetRoles(roleType string) ([]models.Role, error) {
	return s.repo.GetRoles(roleType)
}

func (s *Service) UpdateUserRole(userNo uint, req ChangeUserRoleRequest) (*models.User, error) {
	userRepo := auth.NewRepository(s.repo.db, nil)

	user, err := userRepo.FindByUserNo(userNo)
	if err != nil {
		return nil, errors.New("user not found")
	}

	modifier, err := userRepo.FindByUserNo(req.LastModifiedBy)
	if err != nil {
		return nil, errors.New("last modified by user not found")
	}

	// ✅ FIX: Get old role BEFORE changing and check for errors
	oldRole, err := s.repo.GetRole(user.RoleNo)
	if err != nil {
		return nil, fmt.Errorf("failed to get old role: %w", err)
	}

	// Update user role
	user.RoleNo = req.RoleNo
	user.LastModifiedBy = &req.LastModifiedBy

	newRole, err := s.repo.GetRole(req.RoleNo)
	if err != nil {
		return nil, fmt.Errorf("failed to get new role: %w", err)
	}

	// ✅ FIX: Get permissions and check for errors
	permissions, err := s.repo.GetRolePermissions(req.RoleNo)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}

	var finPermissions []email.PermissionStruct
	for i := 0; i < len(permissions); i++ {
		var perm = email.PermissionStruct{
			ActionPage:        permissions[i].Permission.PageKey,
			ActionName:        permissions[i].Permission.ActionKey,
			ActionDescription: permissions[i].Permission.Description,
		}
		finPermissions = append(finPermissions, perm)
	}

	// ✅ FIX: Update user in database
	if err := s.repo.UpdateUser(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// ✅ FIX: Send email and handle errors properly
	emailService := email.NewEmailService()
	if err := emailService.SendRoleChange(user.Email, oldRole.RoleName, newRole.RoleName, user.UserName, modifier.UserName, finPermissions); err != nil {
		// Log the error but don't fail the entire operation since the role was already updated
		fmt.Printf("Warning: Failed to send role change email to %s: %v\n", user.Email, err)
	}

	return userRepo.FindByUserNo(user.UserNo)
}

func (s *Service) DeleteUser(userNo uint, req DeleteUserRequest) (*models.User, error) {
	userRepo := auth.NewRepository(s.repo.db, nil)

	user, err := userRepo.FindByUserNo(userNo)
	if err != nil {
		return nil, errors.New("user not found")
	}

	user.IsLocked = true
	user.LastModifiedBy = &req.LastModifiedBy

	if err := s.repo.UpdateUser(user); err != nil {
		return nil, err
	}

	return userRepo.FindByUserNo(user.UserNo)
}
