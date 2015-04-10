package main

import (
	"os"
	"os/user"
)

type ProcessConfig struct {
	cmdpath  string
	sockfile *os.File
	user     *user.User
}

func NewProcessConfig(cmdpath string, sockfile *os.File, u *user.User) *ProcessConfig {
	return &ProcessConfig{cmdpath, sockfile, u}
}
