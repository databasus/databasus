package users_repositories

import "sync"

var (
	getUserRepository = sync.OnceValue(func() *UserRepository {
		return &UserRepository{}
	})
	getUsersSettingsRepository = sync.OnceValue(func() *UsersSettingsRepository {
		return &UsersSettingsRepository{}
	})
	getPasswordResetRepository = sync.OnceValue(func() *PasswordResetRepository {
		return &PasswordResetRepository{}
	})
	getTwoFactorRepository = sync.OnceValue(func() *TwoFactorRepository {
		return &TwoFactorRepository{}
	})
)

func GetUserRepository() *UserRepository {
	return getUserRepository()
}

func GetUsersSettingsRepository() *UsersSettingsRepository {
	return getUsersSettingsRepository()
}

func GetPasswordResetRepository() *PasswordResetRepository {
	return getPasswordResetRepository()
}

func GetTwoFactorRepository() *TwoFactorRepository {
	return getTwoFactorRepository()
}
