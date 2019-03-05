package main

import (
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"

	"github.com/getlantern/systray"

	"github.com/xianghuzhao/vboxctl/icon"
)

var (
	wininet, _           = syscall.LoadLibrary("wininet.dll")
	internetSetOption, _ = syscall.GetProcAddress(wininet, "InternetSetOptionW")
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

	mIEProxy := systray.AddMenuItem("Enable IE Proxy", "Enable IE Proxy")
	systray.AddSeparator()
	mRebootPC := systray.AddMenuItem("Reboot PC", "Reboot the PC")
	mShutdownPC := systray.AddMenuItem("Shutdown PC", "Shutdown the PC")
	systray.AddSeparator()
	mStartVM := systray.AddMenuItem("Start VM", "Start the VM")
	mPoweroffVM := systray.AddMenuItem("Poweroff VM", "Poweroff the VM")
	systray.AddSeparator()
	mPoweroffVMAndExit := systray.AddMenuItem("Poweroff VM and Exit", "Poweroff the VM and exit")
	systray.AddSeparator()
	mExit := systray.AddMenuItem("Exit", "Exit the whole app")

	go func() {
		for {
			select {
			case <-mStartVM.ClickedCh:
				systray.SetIcon(startIco)
				logger.Println("Start VM")
				startVM()
			case <-mPoweroffVM.ClickedCh:
				systray.SetIcon(stopIco)
				logger.Println("Poweroff VM")
				poweroffVM()
			case <-mPoweroffVMAndExit.ClickedCh:
				logger.Println("Poweroff VM")
				poweroffVM()
				systray.Quit()
				return
			case <-mExit.ClickedCh:
				systray.Quit()
				return
			case <-mShutdownPC.ClickedCh:
				logger.Println("Poweroff VM")
				poweroffVM()
				time.Sleep(10 * time.Second)
				logger.Println("Shutdown the PC")
				runCmd("cmd", "/C", "shutdown", "/t", "0", "/s")
				systray.Quit()
				return
			case <-mRebootPC.ClickedCh:
				logger.Println("Poweroff VM")
				poweroffVM()
				time.Sleep(10 * time.Second)
				logger.Println("Reboot the PC")
				runCmd("cmd", "/C", "shutdown", "/t", "0", "/r")
				systray.Quit()
				return
			case <-mIEProxy.ClickedCh:
				if mIEProxy.Checked() {
					disableIEProxy()
					mIEProxy.Uncheck()
				} else {
					enableIEProxy()
					mIEProxy.Check()
				}
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

func updateIEOption() {
	ret, _, callErr := syscall.Syscall6(uintptr(internetSetOption),
		4,
		0,
		95,
		0,
		0,
		0,
		0)
	if callErr != 0 {
		log.Print("Call MessageBox", callErr)
	}
	if ret == 0 {
		log.Print("Run InternetSetOptionW error")
	}
	return
}

func enableIEProxy() {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.ALL_ACCESS)
	if err != nil {
		log.Print(err)
		return
	}
	defer key.Close()

	key.SetStringValue("ProxyOverride", "<local>;localhost;127.*;10.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*;192.168.*")
	key.SetStringValue("ProxyServer", "127.0.0.1:3128")
	key.SetDWordValue("ProxyEnable", 1)

	updateIEOption()
}

func disableIEProxy() {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.ALL_ACCESS)
	if err != nil {
		log.Print(err)
		return
	}
	defer key.Close()

	key.SetDWordValue("ProxyEnable", 0)

	updateIEOption()
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
	disableIEProxy()

	systray.Run(onReady, onExit)

	logger.Println("Exit application")
	logger.Println("--------------------------------------------------------------------------------")
}
