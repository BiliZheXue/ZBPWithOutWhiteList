package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/layou233/ZBProxy/config"
	"github.com/layou233/ZBProxy/console"
	"github.com/layou233/ZBProxy/service"
	"github.com/layou233/ZBProxy/version"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
)

func main() {
	log.SetOutput(color.Output)
	console.SetTitle(fmt.Sprintf("ZBProxy %v | Running...", version.Version))
	console.Println(color.HiRedString(` ______  _____   _____   _____    _____  __    __ __    __
|___  / |  _  \ |  _  \ |  _  \  /  _  \ \ \  / / \ \  / /
   / /  | |_| | | |_| | | |_| |  | | | |  \ \/ /   \ \/ /`), color.HiWhiteString(`
  / /   |  _  { |  ___/ |  _  /  | | | |   }  {     \  /
 / /__  | |_| | | |     | | \ \  | |_| |  / /\ \    / /
/_____| |_____/ |_|     |_|  \_\ \_____/ /_/  \_\  /_/`))
	color.HiGreen("Welcome to ZBProxy %s (%s)!\n", version.Version, version.CommitHash)
	color.HiBlack("Build Information: %s, %s/%s\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH)
	go version.CheckUpdate()

	config.LoadConfig()

	for _, s := range config.Config.Services {
		go service.StartNewService(s)
	}

	// hot reload
	// use inotify on Linux
	// use Win32 ReadDirectoryChangesW on Windows
	{
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Panic(err)
		}
		defer watcher.Close()
		err = config.MonitorConfig(watcher)
		if err != nil {
			log.Panic("Config Reload Error : ", err)
		}
	}

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGHUP)
	for {
		// wait for signal
		if <-osSignals == syscall.SIGHUP { // config reload
			log.Println(color.HiMagentaString("Config Reload : SIGHUP signal received. Reloading..."))
			if config.LoadLists(true) { // reload success
				log.Println(color.HiMagentaString("Config Reload : Successfully reloaded Lists."))
			}
		} else { // stop the program
			// sometimes after the program exits on Windows, the ports are still occupied and "listening".
			// so manually closes these listeners when the program exits.
			for _, listener := range service.ListenerArray {
				if listener != nil { // avoid null pointers
					listener.Close()
				}
			}
			break
		}
	}
}
