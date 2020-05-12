// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package cmd

import (
	"fmt"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
)

func setUserToCmd(u string, finalEnv []string, c *exec.Cmd) error {
	userCredential, err := getUserCredential(u)
	if err != nil {
		return err
	}
	c.Env = append(finalEnv)
	c.SysProcAttr = &syscall.SysProcAttr{}
	c.SysProcAttr.Credential = userCredential
	setEnvirons(u, finalEnv, c)
	return nil
}

func setEnvirons(u string, finalEnv []string, c *exec.Cmd) error {
	gottenUser, err := user.Lookup(u)

	if err != nil {
		return err
	}
	c.Env = append(
		[]string{fmt.Sprintf("HOME=%s", gottenUser.HomeDir)}, // This way if HOME can be redefined. Only last remains
		finalEnv...)
	return nil
}

func getUserCredential(u string) (*syscall.Credential, error) {
	gottenUser, err := user.Lookup(u)

	if err != nil {
		return nil, err
	}
	uid, err := strconv.ParseUint(gottenUser.Uid, 10, 32)
	if err != nil {
		return nil, err
	}
	gid, err := strconv.ParseUint(gottenUser.Gid, 10, 32)
	if err != nil {
		return nil, err
	}
	groups := make([]uint32, 0, 0)
	groupIdStrs, err := gottenUser.GroupIds()
	if err != nil {
		return nil, err
	}
	for _, sgid := range groupIdStrs {
		supplementaryGid, err := strconv.ParseUint(sgid, 10, 32)
		if err != nil {
			return nil, err
		}
		groups = append(groups, uint32(supplementaryGid))
	}

	return &syscall.Credential{
		Uid:         uint32(uid),
		Gid:         uint32(gid),
		Groups:      groups,
		NoSetGroups: false,
	}, nil
}
