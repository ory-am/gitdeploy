package job

import (
	"os/exec"
)

func GetLogs(app string) (string, error) {
	o, err := exec.Command("flynn", "-a", app, "log").CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(o), nil
}
