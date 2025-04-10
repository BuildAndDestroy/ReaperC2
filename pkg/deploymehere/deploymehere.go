package deploymehere

import (
	"os"
	"strings"
)

// Check for env variable. This will help with setup flow depending on where we are deployed.
func GetDeploymentEnv() string {
	validEnvs := map[string]bool{
		"AWS":    true,
		"AZURE":  true,
		"GCP":    true,
		"ONPREM": true,
	}
	env := strings.ToUpper(os.Getenv("DEPLOY_ENV"))
	if validEnvs[env] {
		return env
	}
	return "ONPREM"
}
