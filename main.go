package main

import (
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/getlantern/systray"

	"github.com/xianghuzhao/vboxctl/icon"
)

var logger *log.Logger

func onReady() {
	logger.Println("Create systray")

	startIco, err := icon.Asset("start.ico")
	if err != nil {
		logger.Println("Can not access asset start.ico: %s", err)
	}
	stopIco, err := icon.Asset("stop.ico")
	if err != nil {
		logger.Println("Can not access asset stop.ico: %s", err)
	}

	systray.SetIcon(startIco)
	systray.SetTitle("Vboxctl")
	systray.SetTooltip("Virtualbox Control")

	mStart := systray.AddMenuItem("Start", "Start the VM")
	mPoweroff := systray.AddMenuItem("Poweroff", "Poweroff the VM")
	systray.AddSeparator()
	mExit := systray.AddMenuItem("Exit", "Exit the whole app")

	go func() {
		for {
			select {
			case <-mStart.ClickedCh:
				systray.SetIcon(startIco)
				startVM()
			case <-mPoweroff.ClickedCh:
				systray.SetIcon(stopIco)
				poweroffVM()
			case <-mExit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	logger.Println("Quit systray")
}

func startVM() {
	go func() {
		cmd := exec.Command("C:\\Program Files\\Oracle\\VirtualBox\\VBoxManage.exe", "startvm", "Arch", "--type", "headless")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		logger.Println("Start VM")
		err := cmd.Start()
		if err != nil {
			logger.Printf("Run command error: %s", err)
		}
	}()
}

func poweroffVM() {
	go func() {
		cmd := exec.Command("C:\\Program Files\\Oracle\\VirtualBox\\VBoxManage.exe", "controlvm", "Arch", "acpipowerbutton")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		logger.Println("Poweroff VM")
		err := cmd.Start()
		if err != nil {
			logger.Printf("Run command error: %s", err)
		}
	}()
}

func main() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.OpenFile(path.Join(dir, "vboxctl.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	logger = log.New(f, "[VboxCtl] ", log.LstdFlags)

	logger.Println("Start application")
	time.Sleep(30 * time.Second)
	logger.Println("After waiting for 30 seconds")

	startVM()

	// Should be called at the very beginning of main().
	systray.Run(onReady, onExit)
	poweroffVM()

	logger.Println("Exit application")
	logger.Println("--------------------------------------------------------------------------------")
}
