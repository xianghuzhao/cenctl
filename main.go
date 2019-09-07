package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"

	"github.com/getlantern/systray"

	"github.com/xianghuzhao/cenctl/icon"
)

var (
	wininet, _           = syscall.LoadLibrary("wininet.dll")
	internetSetOption, _ = syscall.GetProcAddress(wininet, "InternetSetOptionW")
)

var logger *log.Logger

var configFilename = "config.json"

type proc struct {
	Name      string   `json:"name"`
	Path      string   `json:"path"`
	Args      []string `json:"args"`
	AutoStart bool     `json:"auto_start"`
}

type config struct {
	VMName string `json:"vm_name"`
	Proc   []proc `json:"proc"`
	V2ray  struct {
		Dir        string `json:"dir"`
		ConfigFile string `json:"config_file"`
		Config     []struct {
			Address string `json:"address"`
			Port    int    `json:"port"`
			ID      string `json:"id"`
		} `json:"config"`
	} `json:"v2ray"`
}

var cfg config

var cfgV2ray map[string]interface{}

func (p proc) started() bool {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq "+p.Name)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Printf("Get output for tasklist error: %s\n", err)
	}

	err = cmd.Start()
	if err != nil {
		logger.Printf("Run tasklist command error: %s\n", err)
	}

	started := false

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), p.Name) {
			started = true
			break
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Printf("Reading tasklist output error: %s\n", err)
	}

	err = cmd.Wait()
	if err != nil {
		logger.Printf("Command finished with error: %s\n", err)
	}

	return started
}

func (p proc) start() {
	runCmd(p.Path, p.Args...)
}

func (p proc) stop() {
	runCmdAndWait("taskkill", "/IM", p.Name, "/F")
}

func onReady() {
	logger.Println("Create systray")

	startIco, err := icon.Asset("start.ico")
	if err != nil {
		logger.Printf("Can not access asset start.ico: %s", err)
	}
	stopIco, err := icon.Asset("stop.ico")
	if err != nil {
		logger.Printf("Can not access asset stop.ico: %s", err)
	}

	systray.SetIcon(startIco)
	systray.SetTitle("CenCtl")
	systray.SetTooltip("Center Control")

	var cases []reflect.SelectCase

	mIEProxy := systray.AddMenuItem("Enable IE Proxy", "Enable IE Proxy")
	cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(mIEProxy.ClickedCh)})

	systray.AddSeparator()
	v2rayItemStart := len(cases)

	var mV2rayItems []*systray.MenuItem
	curV2rayItem := currentV2rayConfig()
	for _, v2rayItem := range cfg.V2ray.Config {
		mV2rayItem := systray.AddMenuItem("V2ray: "+v2rayItem.Address, "V2ray: "+v2rayItem.Address)
		cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(mV2rayItem.ClickedCh)})
		mV2rayItems = append(mV2rayItems, mV2rayItem)
		if v2rayItem.Address == curV2rayItem {
			mV2rayItem.Check()
		}
	}

	systray.AddSeparator()
	procItemStart := len(cases)

	var mProcItems []*systray.MenuItem
	for _, p := range cfg.Proc {
		mProcItem := systray.AddMenuItem("Proc: "+p.Name, "Proc: "+p.Name)
		cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(mProcItem.ClickedCh)})
		mProcItems = append(mProcItems, mProcItem)
		if p.AutoStart || p.started() {
			mProcItem.Check()
		}
	}

	systray.AddSeparator()
	poweroffItemStart := len(cases)

	mRebootPC := systray.AddMenuItem("Reboot PC", "Reboot the PC")
	cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(mRebootPC.ClickedCh)})
	mShutdownPC := systray.AddMenuItem("Shutdown PC", "Shutdown the PC")
	cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(mShutdownPC.ClickedCh)})

	systray.AddSeparator()

	mStartVM := systray.AddMenuItem("Start VM", "Start the VM")
	cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(mStartVM.ClickedCh)})
	mPoweroffVM := systray.AddMenuItem("Poweroff VM", "Poweroff the VM")
	cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(mPoweroffVM.ClickedCh)})

	systray.AddSeparator()

	mPoweroffVMAndExit := systray.AddMenuItem("Poweroff VM and Exit", "Poweroff the VM and exit")
	cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(mPoweroffVMAndExit.ClickedCh)})

	systray.AddSeparator()

	mExit := systray.AddMenuItem("Exit", "Exit the whole app")
	cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(mExit.ClickedCh)})

	go func() {
		for {
			chosen, _, _ := reflect.Select(cases)

			switch {
			case chosen == 0:
				if mIEProxy.Checked() {
					disableIEProxy()
					logger.Println("IE proxy disabled")
					mIEProxy.Uncheck()
				} else {
					enableIEProxy()
					logger.Println("IE proxy enabled")
					mIEProxy.Check()
				}
			case chosen >= v2rayItemStart && chosen < procItemStart:
				for i, v2rayItem := range cfg.V2ray.Config {
					mV2rayItem := mV2rayItems[i]
					if i+v2rayItemStart == chosen {
						if !mV2rayItem.Checked() {
							logger.Printf("Switch v2ray to \"%s\"\n", v2rayItem.Address)
							switchV2ray(v2rayItem.Address, v2rayItem.Port, v2rayItem.ID)
							mV2rayItem.Check()
						}
					} else {
						if mV2rayItem.Checked() {
							mV2rayItem.Uncheck()
						}
					}
				}
			case chosen >= procItemStart && chosen < poweroffItemStart:
				p := cfg.Proc[chosen-procItemStart]
				mProcItem := mProcItems[chosen-procItemStart]
				if mProcItem.Checked() {
					logger.Printf("Stop proc \"%s\"\n", p.Name)
					p.stop()
					mProcItem.Uncheck()
				} else {
					logger.Printf("Start proc \"%s\"\n", p.Name)
					p.start()
					mProcItem.Check()
				}
			case chosen == poweroffItemStart:
				logger.Println("Poweroff VM")
				poweroffVM()
				time.Sleep(10 * time.Second)
				logger.Println("Reboot the PC")
				runCmd("cmd", "/C", "shutdown", "/t", "0", "/r")
				systray.Quit()
				return
			case chosen == poweroffItemStart+1:
				logger.Println("Poweroff VM")
				poweroffVM()
				time.Sleep(10 * time.Second)
				logger.Println("Shutdown the PC")
				runCmd("cmd", "/C", "shutdown", "/t", "0", "/s")
				systray.Quit()
				return
			case chosen == poweroffItemStart+2:
				systray.SetIcon(startIco)
				logger.Println("Start VM")
				startVM()
			case chosen == poweroffItemStart+3:
				systray.SetIcon(stopIco)
				logger.Println("Poweroff VM")
				poweroffVM()
			case chosen == poweroffItemStart+4:
				logger.Println("Poweroff VM")
				poweroffVM()
				systray.Quit()
				return
			case chosen == poweroffItemStart+5:
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

func runCmdAndWait(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err := cmd.Start()
	if err != nil {
		logger.Printf("Run command error: %s\n", err)
	}
	err = cmd.Wait()
	if err != nil {
		logger.Printf("Command finished with error: %s\n", err)
	}
}

func startVM() {
	runCmd("C:\\Program Files\\Oracle\\VirtualBox\\VBoxManage.exe", "startvm", cfg.VMName, "--type", "headless")
}

func poweroffVM() {
	runCmd("C:\\Program Files\\Oracle\\VirtualBox\\VBoxManage.exe", "controlvm", cfg.VMName, "acpipowerbutton")
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
		log.Print("Call InternetSetOption", callErr)
	}
	if ret == 0 {
		log.Print("Run InternetSetOption error")
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

func startV2ray() {
	runCmd(path.Join(cfg.V2ray.Dir, "wv2ray.exe"), "-config", path.Join(cfg.V2ray.Dir, cfg.V2ray.ConfigFile))
}

func stopV2ray() {
	runCmdAndWait("taskkill", "/IM", "wv2ray.exe", "/F")
}

func switchV2ray(address string, port int, id string) {
	go func() {
		stopV2ray()

		cfgVnext := cfgV2ray["outbounds"].([]interface{})[0].(map[string]interface{})["settings"].(map[string]interface{})["vnext"].([]interface{})[0].(map[string]interface{})
		cfgVnext["address"] = address
		cfgVnext["port"] = port
		cfgVnext["users"].([]interface{})[0].(map[string]interface{})["id"] = id

		saveV2rayConfig()

		time.Sleep(time.Second)
		startV2ray()
	}()
}

func loadV2rayConfig() {
	buffer, err := ioutil.ReadFile(path.Join(cfg.V2ray.Dir, cfg.V2ray.ConfigFile))
	if err != nil {
		logger.Panicf("V2ray config file \"%s\" read error: %s\n", cfg.V2ray.ConfigFile, err)
	}

	err = json.Unmarshal(buffer, &cfgV2ray)
	if err != nil {
		logger.Panicf("Parse v2ray config error: %s\n", err)
	}
}

func saveV2rayConfig() {
	data, err := json.MarshalIndent(&cfgV2ray, "", "  ")
	if err != nil {
		logger.Printf("Save v2ray config error: %s\n", err)
	}

	err = ioutil.WriteFile(path.Join(cfg.V2ray.Dir, cfg.V2ray.ConfigFile), data, 0644)
	if err != nil {
		logger.Panicf("V2ray config file \"%s\" write error: %s\n", cfg.V2ray.ConfigFile, err)
	}
}

func currentV2rayConfig() string {
	return cfgV2ray["outbounds"].([]interface{})[0].(map[string]interface{})["settings"].(map[string]interface{})["vnext"].([]interface{})[0].(map[string]interface{})["address"].(string)
}

func loadConfig(dir string) {
	buffer, err := ioutil.ReadFile(path.Join(dir, configFilename))
	if err != nil {
		logger.Panicf("Config file \"%s\" read error: %s\n", configFilename, err)
	}

	err = json.Unmarshal(buffer, &cfg)
	if err != nil {
		logger.Panicf("Parse config error: %s\n", err)
	}
}

func autoStart() {
	for _, p := range cfg.Proc {
		if !p.AutoStart {
			continue
		}
		if p.started() {
			logger.Printf("Command already started: %s\n", p.Name)
			continue
		}
		logger.Printf("Auto start command: %s\n", p.Name)
		p.start()
	}
}

func main() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.OpenFile(path.Join(dir, "cenctl.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	logger = log.New(f, "[CenCtl] ", log.LstdFlags)

	logger.Println("================================================================================")
	logger.Println("Start application")

	loadConfig(dir)

	loadV2rayConfig()

	go func() {
		autoStart()
	}()

	go func() {
		time.Sleep(30 * time.Second)
		logger.Println("After waiting for 10 seconds")

		logger.Println("Start VM")
		startVM()
	}()

	disableIEProxy()

	systray.Run(onReady, onExit)

	logger.Println("Exit application")
	logger.Println("--------------------------------------------------------------------------------")
}
