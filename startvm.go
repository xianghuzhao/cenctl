package main

import (
	"fmt"
	"os/exec"
	"syscall"
	//"time"
)

func main() {
	//time.Sleep(30 * time.Second)
	cmd := exec.Command("C:\\Program Files\\Oracle\\VirtualBox\\VBoxManage.exe", "startvm", "Arch", "--type", "headless")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err := cmd.Start()
	if err != nil {
		fmt.Printf("Run command error: %s", err)
	}
}
