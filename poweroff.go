package main

import (
	"fmt"
	"os/exec"
	"syscall"
)

func main() {
	cmd := exec.Command("C:\\Program Files\\Oracle\\VirtualBox\\VBoxManage.exe", "controlvm", "Arch", "acpipowerbutton")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err := cmd.Start()
	if err != nil {
		fmt.Printf("Run command error: %s", err)
	}
}
