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

	mShutdown := systray.AddMenuItem("Shutdown", "Shutdown the system")
	systray.AddSeparator()
	mStart := systray.AddMenuItem("Start", "Start the VM")
	mPoweroff := systray.AddMenuItem("Poweroff", "Poweroff the VM")
	systray.AddSeparator()
	mPoweroffAndExit := systray.AddMenuItem("Poweroff and Exit", "Poweroff the VM and exit")
	systray.AddSeparator()
	mExit := systray.AddMenuItem("Exit", "Exit the whole app")

	go func() {
		for {
			select {
			case <-mStart.ClickedCh:
				systray.SetIcon(startIco)
				logger.Println("Start VM")
				startVM()
			case <-mPoweroff.ClickedCh:
				systray.SetIcon(stopIco)
				logger.Println("Poweroff VM")
				poweroffVM()
			case <-mPoweroffAndExit.ClickedCh:
				logger.Println("Poweroff VM")
				poweroffVM()
				systray.Quit()
				return
			case <-mExit.ClickedCh:
				systray.Quit()
				return
			case <-mShutdown.ClickedCh:
				logger.Println("Poweroff VM")
				poweroffVM()
				time.Sleep(10 * time.Second)
				logger.Println("Shutdown the system")
				runCmd("cmd", "/C", "shutdown", "/t", "0", "/s")
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	logger.Println("Quit systray")
}

func runCmd(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err := cmd.Start()
	if err != nil {
		logger.Printf("Run command error: %s", err)
	}
}

func startVM() {
	go func() {
		runCmd("C:\\Program Files\\Oracle\\VirtualBox\\VBoxManage.exe", "startvm", "Arch", "--type", "headless")
	}()
}

func poweroffVM() {
	go func() {
		runCmd("C:\\Program Files\\Oracle\\VirtualBox\\VBoxManage.exe", "controlvm", "Arch", "acpipowerbutton")
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

	logger.Println("================================================================================")
	logger.Println("Start application")
	time.Sleep(10 * time.Second)
	logger.Println("After waiting for 10 seconds")

	logger.Println("Start VM")
	startVM()

	systray.Run(onReady, onExit)

	logger.Println("Exit application")
	logger.Println("--------------------------------------------------------------------------------")
}
