package users_services

import (
	"sync"
	"sync/atomic"

	"databasus-backend/internal/features/email"
	users_repositories "databasus-backend/internal/features/users/repositories"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/ratelimiter"
)

var getUserService = sync.OnceValue(func() *UserService {
	return &UserService{
		users_repositories.GetUserRepository(),
		encryption.GetMasterKey(),
		getSettingsService(),
		nil,
		email.GetInstanceSender(),
		users_repositories.GetPasswordResetRepository(),
		users_repositories.GetTwoFactorRepository(),
		ratelimiter.GetCounter(),
		logger.GetLogger(),
	}
})

var getSettingsService = sync.OnceValue(func() *SettingsService {
	return &SettingsService{
		users_repositories.GetUsersSettingsRepository(),
		users_repositories.GetUserRepository(),
		nil,
		email.GetInstanceSender(),
	}
})

var getSignInCodeBackgroundService = sync.OnceValue(func() *SignInCodeBackgroundService {
	return &SignInCodeBackgroundService{
		getUserService(),
		logger.GetLogger(),
		atomic.Bool{},
	}
})

var getManagementService = sync.OnceValue(func() *UserManagementService {
	return &UserManagementService{
		users_repositories.GetUserRepository(),
		nil,
	}
})

func GetUserService() *UserService {
	return getUserService()
}

func GetSettingsService() *SettingsService {
	return getSettingsService()
}

func GetManagementService() *UserManagementService {
	return getManagementService()
}

func GetSignInCodeBackgroundService() *SignInCodeBackgroundService {
	return getSignInCodeBackgroundService()
}
