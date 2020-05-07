package cmd

import (
	"log"
	"os/exec"
)

func setUserToCmd(u string, finalEnv []string, c *exec.Cmd) error {
	log.Print("On windows nothing is done when user is set...")
	return nil
}
