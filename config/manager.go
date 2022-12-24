package config

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"github.com/layou233/ZBProxy/common/set"
	"github.com/layou233/ZBProxy/version"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
)

const DefaultMotd = `data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAEAAAABACAMAAACdt4HsAAAAAXNSR0IB2cksfwAAAAlwSFlzAAALEwAACxMBAJqcGAAAAeZQTFRF/+IAzrcHKyodHx8fR0IZ89cC2MAFKikdw64IIyMeXFQX/eAAxa8IIiIfrpwLISAfgHQRnYwOuKQKaF4VJyYecGUUj4AQ99sBUUsYMS4d38YE+98AWVEXrJkM7dMCQDsaPzsb9NgCTkgYuqYJ3sYEUEoYODQcJiUe0LkHW1MX+94B178GMC4dcmgTwq0I5s0DpJINoZAN69shjLi/i7i/ZZfEU4fHVIfHeqnB0NFOZqr/f7PWY1sVtcd7n7+g+NwBm76nv8tq7tMCRkEagLPU4Nc06M4DPjob4McE49cvgLTUxMxjnL6lpMCYi6B7s8d/ssZ/a4iJQmWPZaj7NUdcXnNvYJ7sMkVcu8pxW5XdLjtMNjMcssaBVovNKTI9qMORUYG+JCgtnr6hTHiu+N8MlbuxR26f7twci7fBQ2SPfXES5dgsgbTRPVuA29U8ebLgxMxdiHoQ0tFLb63w0dFMOFFverHe3NU6PVl9hLTN5tkpQWKLj7m78d0XRWqZmb2q++EGSnOnICIkTny0rsSHqpgMJSoxUoTCuch1KTNAV43QZHlxSXKkcY6IdrDl+N4EZqn8lLuy2tQ+t8h40tJMmLyr8N0ZtqIKerHdd7DjloYPkbq3dGkT59ooq8OMxM1ixs1gzbYH1nu7OAAAA1JJREFUeJyFl/dfE0EQxZesRkPU0CIlGCACIiAWrBhFQQELotjF3sGK2HvD3huK2P5TL7nL7bwZPnvvR/bNl83eu5k9pajyQpppylQwhKdxg0JNj3BDfhQMM2baAWpWjBMK7AYOUIUcoAutBgFQBRwQK7IZJEAVc0JJHqxH4wCYLQClZZxQXgGGRCUF6DmCkKzihOokGGpSFBCbKwi15ZxQh3FI1hOAntcgCHklnDC/EQxNIQLQzWFBKBJxWICGlggB6PqFgiDjwE57EQXwxGck4qAXT2pwATzxS1pbly5bzrRiZasn1xSnAJb4VW02rXZNiTQFYOLXWAFrPVd7igIg8eusgPU5W0cnAUDiN1gBG31fVzcB0MRb69vIVnuaCcAkfpO1fjM9rQYK8BO/xQrYinmArHiJ77UCtlkAXuL7iLbv6O+H+p27WiwAnnhHid1Qv2evDtVaADzxSu2D+v0HHM9AjwXA+8tBqD90OGvq7jCGOk7A/nIE6o8e80ypGt+RrOYE2l+OQ/0JY6pM+J4K0QAHurylk6eg/jQ1pU0DkAOz0/2JZ85C/SCahsw+xcDUqXbnz+fOQ/0FbiJPXE7U4qi6eAnqL6e5hz5x2UKHr4zQ8pGrbvsARUgkRQu9Bv/++g3ltg9UqMkH4MDU+ibU37qdNTntg6nejDwYmPoO1N+955l6mjmhrNQn0IF5H+ofPPRNDeKJV5pjMAPzEdT30mTLJ04WvYH5+AnUP1WgUQvAHZjPnkP9C8U0bAFkBubLV1D/OsoB0WILwInDG6gf7H/bx8RvQIwfxw1IveOBYoD3HwIAHx1T1wAF4I/8FFDf1pdxhUmg1DAAPgcBvmRtJFBKj1LA1yDAN9c3FjMA+lKq7wH1P3LGcQOAORG0gZ++c8IAyJz4FQT47QMa8w3AzIk/QYC/ZrPe6+tOJjInXPHEaz2uuDq6DYDOiax44h2NCUK2v+RmY5q9NfLOX/JPEDJx8IfrEFuULbS8VhCc/mKmM78ZQOKzqpIX4ln0lsZvBmHRQrMDCzVOAJEWtihbKD+pjMgymROuxsTI4yeFADonvA1ywCR3KFgmc8LVhCCIOxQu80DlEm8kvtHYepwdU42Yyfwbja+zb20v8VT4jfYfTXskk4+wbR0AAAAASUVORK5CYII=`

var (
	Config               configMain
	Lists                map[string]*set.StringSet
	GuildPlayerNameLists map[string]*set.StringSet
	reloadLock           sync.Mutex
)

func LoadConfig() {
	configFile, err := os.ReadFile("ZBProxy.json")
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("Configuration file is not exists. Generating a new one...")
			generateDefaultConfig()
			goto success
		} else {
			log.Panic(color.HiRedString("Unexpected error when loading config: %s", err.Error()))
		}
	}

	err = json.Unmarshal(configFile, &Config)
	if err != nil {
		log.Panic(color.HiRedString("Config format error: %s", err.Error()))
	}

success:
	LoadLists(false)
	log.Println(color.HiYellowString("Successfully loaded config from file."))
}

func generateDefaultConfig() {
	file, err := os.Create("ZBProxy.json")
	if err != nil {
		log.Panic("Failed to create configuration file:", err.Error())
	}
	Config = configMain{
		Services: []*ConfigProxyService{
			{
				Name:          "HypixelDefault",
				TargetAddress: "mc.hypixel.net",
				TargetPort:    25565,
				Listen:        25565,
				Flow:          "auto",
				Minecraft: minecraft{
					EnableHostnameRewrite: true,
					OnlineCount: onlineCount{
						Max:            114514,
						Online:         -1,
						EnableMaxLimit: false,
					},
					MotdFavicon:     "{DEFAULT_MOTD}",
					MotdDescription: "§d{NAME}§e service is working on §a§o{INFO}§r\n§c§lProxy for §6§n{HOST}:{PORT}§r",
				},
			},
		},
		Lists: map[string][]string{
			//"test": {"foo", "bar"},
		},
	}
	newConfig, _ := json.MarshalIndent(Config, "", "    ")
	_, err = file.WriteString(strings.ReplaceAll(string(newConfig), "\n", "\r\n"))
	file.Close()
	if err != nil {
		log.Panic("Failed to save configuration file:", err.Error())
	}
}

func LoadLists(isReload bool) bool {
	reloadLock.Lock()
	defer reloadLock.Unlock()
	if isReload {
		configFile, err := os.ReadFile("ZBProxy.json")
		if err != nil {
			if os.IsNotExist(err) {
				log.Println(color.HiRedString("Fail to reload : Configuration file is not exists."))
			} else {
				log.Println(color.HiRedString("Unexpected error when reloading config: %s", err.Error()))
			}
			return false
		}

		err = json.Unmarshal(configFile, &Config)
		if err != nil {
			log.Println(color.HiRedString("Fail to reload : Config format error: %s", err.Error()))
			return false
		}
	}
	// log.Println("Lists:", Config.Lists)
	if l := len(Config.Lists); l == 0 { // if nothing in Lists
		Lists = map[string]*set.StringSet{} // empty map
	} else {
		Lists = make(map[string]*set.StringSet, l) // map size init
		for k, v := range Config.Lists {
			// log.Println("List: Loading", k, "value:", v)
			list := set.NewStringSetFromSlice(v)
			Lists[k] = &list
		}
	}
	Config.Lists = nil // free memory

	GuildPlayerNameLists = map[string]*set.StringSet{}
	for _, v := range Config.Services {
		if v.Minecraft.HypixelGuild.SampleName != "" && v.Minecraft.HypixelGuild.ApiKey != "" {
			err, l := getHypixelGuildList([]string{v.Minecraft.HypixelGuild.SampleName, v.Minecraft.HypixelGuild.ApiKey})
			if err != nil {
				log.Panic(err)
			}
			list := set.NewStringSetFromSlice(l)
			GuildPlayerNameLists[v.Name] = &list
		}
	}

	for _, s := range Config.Services {
		if s.Minecraft.MotdFavicon == "{DEFAULT_MOTD}" {
			s.Minecraft.MotdFavicon = DefaultMotd
		}
		s.Minecraft.MotdDescription = strings.NewReplacer(
			"{INFO}", "ZBProxy "+version.Version,
			"{NAME}", s.Name,
			"{HOST}", s.TargetAddress,
			"{PORT}", strconv.Itoa(int(s.TargetPort)),
		).Replace(s.Minecraft.MotdDescription)
	}

	debug.FreeOSMemory()
	return true
}

func MonitorConfig(watcher *fsnotify.Watcher) error {
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op.Has(fsnotify.Write) { // config reload
					log.Println(color.HiMagentaString("Config Reload : File change detected. Reloading..."))
					if LoadLists(true) { // reload success
						log.Println(color.HiMagentaString("Config Reload : Successfully reloaded Lists."))
					} else {
						log.Println(color.HiMagentaString("Config Reload : Failed to reload Lists."))
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println(color.HiRedString("Config Reload Error : ", err))
			}
		}
	}()
	return watcher.Add("ZBProxy.json")
}

func getHypixelGuildList(info []string) (err error, NameList []string) {
	cli := http.DefaultClient
	req, err := http.NewRequest("POST", "https://api.mojang.com/profiles/minecraft", strings.NewReader("[\""+info[0]+"\"]"))
	if err != nil {
		return err, nil
	}
	resp, err := cli.Do(req)
	if err != nil {
		return err, nil
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err, nil
	}
	configUuid := make([]*mojangUuidSingle, 0)
	if err = json.Unmarshal(body, &configUuid); err != nil {
		return err, nil
	}
	uuid := configUuid[0].Id
	req, err = http.NewRequest("GET", "https://api.hypixel.net/guild?key="+info[1]+"&player="+uuid, nil)
	if err != nil {
		return err, nil
	}
	resp, err = cli.Do(req)
	if err != nil {
		return err, nil
	}
	body, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err, nil
	}
	var configHypixel hypixelConfig
	var configProfile mojangUuidSingle
	if err = json.Unmarshal(body, &configHypixel); err != nil {
		return err, nil
	}
	if !configHypixel.State {
		return errors.New("wrong apikey"), nil
	}
	log.Println(color.HiYellowString("Loading guild list..."))
	for _, v := range configHypixel.Guild.Members {
		req, err := http.NewRequest("GET", "https://sessionserver.mojang.com/session/minecraft/profile/"+v.Uuid, nil)
		if err != nil {
			continue
		}
		resp, err := cli.Do(req)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}
		err = json.Unmarshal(body, &configProfile)
		if err != nil {
			continue
		}
		NameList = append(NameList, configProfile.Name)
	}
	log.Println(color.HiYellowString("Loaded successfully."))
	return nil, NameList
}
