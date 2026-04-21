package auth

import (
	"eclaim-workshop-deck-api/internal/domain/email"
	"eclaim-workshop-deck-api/internal/domain/settings"
	"eclaim-workshop-deck-api/internal/models"
	"eclaim-workshop-deck-api/pkg/utils"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo      *Repository
	jwtSecret string
}

func NewService(repo *Repository, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

func createEmailService() *email.EmailService {
	return email.NewEmailService()
}

// ---------------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------------

func (s *Service) Register(req RegisterRequest) (*models.User, string, string, error) {
	_, err := s.repo.FindByEmail(req.Email)
	if err == nil {
		return nil, "", "", errors.New("user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", "", err
	}

	user := &models.User{
		RoleNo:    req.RoleNo,
		UserName:  req.Name,
		UserId:    req.UserId,
		Email:     req.Email,
		Password:  string(hashedPassword),
		CreatedBy: &req.CreatedBy,
	}

	if req.UserProfileNo != 0 {
		user.UserProfileNo = &req.UserProfileNo
	}
	if err := s.repo.Create(user); err != nil {
		return nil, "", "", err
	}

	accessToken, err := utils.GenerateToken(user.UserNo, s.jwtSecret)
	if err != nil {
		return nil, "", "", err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.UserNo, s.jwtSecret)
	if err != nil {
		return nil, "", "", err
	}

	return user, accessToken, refreshToken, nil
}

func (s *Service) Login(req LoginRequest) (*models.User, string, error) {
	// 1. Try primary DB first.
	user, err := s.repo.FindByEmail(req.Email)
	if err != nil {
		// 2. Fallback to secondary (App B) DB.
		user, err = s.repo.FindByEmailInAltDB(req.Email)
		if err != nil {
			return nil, "", errors.New("invalid credentials")
		}
	}

	// 3. Verify password.
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, "", errors.New("invalid credentials")
	}

	// 4. Generate a 6-digit OTP.
	otp, err := generateOTP()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate OTP: %w", err)
	}

	// 5. Hash the OTP before storing.
	otpHash, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	// 6. Persist token (valid for 5 minutes).
	expiry := time.Now().Add(5 * time.Minute)
	tokenRecord := &models.UserToken{
		UserNo:     user.UserNo,
		UserToken:  string(otpHash),
		ExpiryDate: expiry,
		CreatedBy:  user.UserNo,
	}
	if err := s.repo.CreateUserToken(tokenRecord); err != nil {
		return nil, "", fmt.Errorf("failed to save OTP: %w", err)
	}

	emailService := createEmailService()
	if err := emailService.Send2FA(req.Email, user.UserName, otp); err != nil {
		fmt.Printf("Warning: failed to send 2FA email to %s: %v\n", req.Email, err)
		// Don't return the error — the OTP is still saved in DB,
		// and dev_otp in the response is your fallback for now.
	}

	return user, otp, nil
}

func (s *Service) VerifyTwoFactor(req VerifyTwoFactorRequest) (*models.User, string, string, error) {
	// 1. Load the pending token row.
	tokenRecord, err := s.repo.FindValidToken(req.UserNo)
	if err != nil {
		return nil, "", "", errors.New("OTP not found or expired — please login again")
	}

	// 2. Compare supplied OTP against stored hash.
	if err := bcrypt.CompareHashAndPassword([]byte(tokenRecord.UserToken), []byte(req.Token)); err != nil {
		return nil, "", "", errors.New("invalid OTP")
	}

	// 3. Load the full user record.
	user, err := s.repo.FindByUserNo(req.UserNo)
	if err != nil {
		return nil, "", "", errors.New("user not found")
	}

	// 4. Issue JWT pair.
	accessToken, err := utils.GenerateToken(user.UserNo, s.jwtSecret)
	if err != nil {
		return nil, "", "", err
	}
	refreshToken, err := utils.GenerateRefreshToken(user.UserNo, s.jwtSecret)
	if err != nil {
		return nil, "", "", err
	}

	return user, accessToken, refreshToken, nil
}

func (s *Service) RefreshToken(req RefreshTokenRequest) (string, string, error) {
	claims, err := utils.ValidateToken(req.RefreshToken, s.jwtSecret)
	if err != nil {
		return "", "", errors.New("invalid or expired refresh token")
	}

	_, err = s.repo.FindByUserNo(claims.UserNo)
	if err != nil {
		return "", "", errors.New("user not found")
	}

	newAccessToken, err := utils.GenerateToken(claims.UserNo, s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	newRefreshToken, err := utils.GenerateRefreshToken(claims.UserNo, s.jwtSecret)
	if err != nil {
		return "", "", err
	}

	return newAccessToken, newRefreshToken, nil
}

func (s *Service) GetUserByEmail(req FindByEmailRequest) (*models.User, error) {
	user, err := s.repo.FindByEmail(req.Email)
	if err != nil {
		return nil, errors.New("user with that email not found")
	}
	return user, nil
}

func (s *Service) ChangePassword(req ChangePasswordRequest) (*models.User, error) {
	user, err := s.repo.FindByEmail(req.Email)
	if err != nil {
		return nil, errors.New("user with that email not found")
	}

	if user.UserId != req.Username {
		return nil, errors.New("invalid username")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid old password")
	}

	if req.NewPassword != req.ConfirmPassword {
		return nil, errors.New("new password and confirmation do not match")
	}

	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user.Password = string(hashedNewPassword)
	user.LastModifiedBy = &user.UserNo

	if err := s.repo.ChangePassword(user); err != nil {
		return nil, err
	}

	emailService := createEmailService()
	_ = emailService.SendChangedPassword(req.Email, user.UserName)

	return user, nil
}

func (s *Service) UpdateAccount(req UpdateAccountRequest) (*models.User, error) {
	userNo := req.UserNo
	var toEmail, newUID string
	var emailChanged, usernameChanged, passwordChanged bool

	user, err := s.repo.FindByUserNo(userNo)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if req.UserNo != user.UserNo {
		return nil, errors.New("unauthorized: you can only update your own account")
	}

	if req.Password != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			return nil, errors.New("invalid old password")
		}
	}

	if req.NewPassword != req.ConfirmPassword {
		return nil, errors.New("new password and confirmation do not match")
	}

	hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		passwordChanged = false
		return nil, err
	}
	passwordChanged = true

	user.Password = string(hashedNewPassword)
	user.LastModifiedBy = &user.UserNo

	if req.Email != "" {
		user.Email = req.Email
		toEmail = req.Email
		emailChanged = true
	} else {
		toEmail = user.Email
		emailChanged = false
	}

	if req.Username != "" {
		user.UserId = req.Username
		newUID = req.Username
	} else {
		newUID = user.UserId
	}

	_ = usernameChanged

	if err := s.repo.UpdateAccount(user); err != nil {
		return nil, err
	}

	emailService := createEmailService()
	_ = emailService.SendUpdatedAccount(toEmail, user.UserName, newUID, emailChanged, usernameChanged, passwordChanged)

	return user, nil
}

func (s *Service) ResetPassword(req ResetPasswordRequest) error {
	user, err := s.repo.FindByEmailAndUsername(req.Email, req.Username)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	newPassword, hashed, err := utils.GenerateRandomPassword(32)
	if err != nil {
		return fmt.Errorf("failed to generate password: %v", err)
	}

	if err := s.repo.UpdatePassword(user.UserNo, string(hashed)); err != nil {
		return fmt.Errorf("failed to update password: %v", err)
	}

	emailService := email.NewEmailService()
	if err := emailService.SendResetEmail(req.Email, req.Username, newPassword); err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

func (s *Service) GetWorkshopDetails(userProfileNo uint) (*models.WorkshopDetails, error) {
	settingsRepo := settings.NewRepository(s.repo.db)
	settingsService := settings.NewService(settingsRepo)
	return settingsService.GetWorkshopDetailsFromUserProfileNo(userProfileNo)
}
