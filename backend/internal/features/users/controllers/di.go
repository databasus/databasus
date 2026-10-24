package users_controllers

import (
	"sync"

	users_services "databasus-backend/internal/features/users/services"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/ratelimiter"
)

var getUserController = sync.OnceValue(func() *UserController {
	return &UserController{
		users_services.GetUserService(),
		ratelimiter.GetCounter(),
		logger.GetLogger(),
	}
})

var getSettingsController = sync.OnceValue(func() *SettingsController {
	return &SettingsController{
		users_services.GetSettingsService(),
		ratelimiter.GetCounter(),
		logger.GetLogger(),
	}
})

var getManagementController = sync.OnceValue(func() *ManagementController {
	return &ManagementController{
		users_services.GetManagementService(),
	}
})

func GetUserController() *UserController {
	return getUserController()
}

func GetSettingsController() *SettingsController {
	return getSettingsController()
}

func GetManagementController() *ManagementController {
	return getManagementController()
}
