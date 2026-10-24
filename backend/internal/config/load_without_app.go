//go:build !testhelpers

package config

func loadWithoutApp() *EnvVariables {
	panic("config.GetEnv is called before app.New installed the configuration")
}
