package service

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/layou233/ZBProxy/common"
	"github.com/layou233/ZBProxy/common/set"
	"github.com/layou233/ZBProxy/config"
	"github.com/layou233/ZBProxy/service/access"
	"github.com/layou233/ZBProxy/service/minecraft"
	"github.com/layou233/ZBProxy/service/transfer"
	"github.com/layou233/ZBProxy/version"
	"log"
	"net"
	"strconv"
	"strings"
)

var ListenerArray = make([]net.Listener, 1)

func StartNewService(s *config.ConfigProxyService) {
	// Check Settings
	var isMinecraftHandleNeeded = s.Minecraft.EnableHostnameRewrite ||
		s.Minecraft.EnableAnyDest ||
		s.Minecraft.EnableMojangCapeRequirement ||
		s.Minecraft.MotdDescription != "" ||
		s.Minecraft.MotdFavicon != ""
	flowType := getFlowType(s.Flow)
	if s.Debug {
		log.Printf("Flow mode : %s", s.Flow)
	}
	if flowType == -1 {
		log.Panic(color.HiRedString("Service %s: Unknown flow type '%s'.", s.Name, s.Flow))
	}
	if s.Minecraft.MotdFavicon == "{DEFAULT_MOTD}" {
		s.Minecraft.MotdFavicon = minecraft.DefaultMotd
	}
	s.Minecraft.MotdDescription = strings.NewReplacer(
		"{INFO}", "ZBProxy "+version.Version,
		"{NAME}", s.Name,
		"{HOST}", s.TargetAddress,
		"{PORT}", strconv.Itoa(int(s.TargetPort)),
	).Replace(s.Minecraft.MotdDescription)
	if s.Minecraft.EnableHostnameRewrite && s.Minecraft.RewrittenHostname == "" {
		s.Minecraft.RewrittenHostname = s.TargetAddress
	}
	listen, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   nil, // listens on all available IP addresses of the local system
		Port: int(s.Listen),
	})
	if err != nil {
		log.Panic(color.HiRedString("Service %s: Can't start listening on port %v: %v", s.Name, s.Listen, err.Error()))
	}
	ListenerArray = append(ListenerArray, listen) // add to ListenerArray
	remoteAddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", s.TargetAddress, s.TargetPort))

	// load access lists
	//var ipAccessLists []*set.StringSet = nil
	ipAccessMode := access.ParseAccessMode(s.IPAccess.Mode)
	if ipAccessMode != access.DefaultMode { // IP access control enabled
		if s.IPAccess.ListTags == nil {
			log.Panic(color.HiRedString("Service %s: ListTags can't be null when access control enabled.", s.Name))
		}
		/*ipAccessLists = make([]*set.StringSet, len(s.IPAccess.ListTags))
		for i := 0; i < len(s.IPAccess.ListTags); i++ {
			ipAccessLists[i], err = access.GetTargetList(s.IPAccess.ListTags[i])
			if err != nil {
				log.Panic(color.HiRedString("Service %s: %s", s.Name, err.Error()))
			}
		}*/
		for _, tag := range s.IPAccess.ListTags {
			if common.GetSecond[error](access.GetTargetList(tag)) != nil {
				log.Panic(color.HiRedString("Service %s: %s", s.Name, err.Error()))
			}
		}
	}

	// load Minecraft player name access lists
	//var mcNameAccessLists []*set.StringSet = nil
	mcNameAccessMode := access.ParseAccessMode(s.Minecraft.NameAccess.Mode)
	if isMinecraftHandleNeeded && mcNameAccessMode != access.DefaultMode { // IP access control enabled
		if s.Minecraft.NameAccess.ListTags == nil {
			log.Panic(color.HiRedString("Service %s: ListTags can't be null when access control enabled.", s.Name))
		}
		/*mcNameAccessLists = make([]*set.StringSet, len(s.Minecraft.NameAccess.ListTags))
		for i := 0; i < len(s.Minecraft.NameAccess.ListTags); i++ {
			mcNameAccessLists[i], err = access.GetTargetList(s.Minecraft.NameAccess.ListTags[i])
			if err != nil {
				log.Panic(color.HiRedString("Service %s: %s", s.Name, err.Error()))
			}
		}*/
		for _, tag := range s.Minecraft.NameAccess.ListTags {
			if common.GetSecond[error](access.GetTargetList(tag)) != nil {
				log.Panic(color.HiRedString("Service %s: %s", s.Name, err.Error()))
			}
		}
	}

	for {
		conn, err := listen.AcceptTCP()
		if err == nil {
			if ipAccessMode != access.DefaultMode {
				// https://stackoverflow.com/questions/29687102/how-do-i-get-a-network-clients-ip-converted-to-a-string-in-golang
				ip := conn.RemoteAddr().(*net.TCPAddr).IP.String()
				hit := false
				for _, list := range s.IPAccess.ListTags {
					if hit = common.Must[*set.StringSet](access.GetTargetList(list)).Has(ip); hit {
						break
					}
				}
				switch ipAccessMode {
				case access.AllowMode:
					if !hit {
						forciblyCloseTCP(conn)
						continue
					}
				case access.BlockMode:
					if hit {
						forciblyCloseTCP(conn)
						continue
					}
				}
			}
			go newConnReceiver(s, conn, isMinecraftHandleNeeded, flowType, remoteAddr, mcNameAccessMode)
		}
	}
}

func getFlowType(flow string) int {
	switch flow {
	case "origin":
		return transfer.FLOW_ORIGIN
	case "linux-zerocopy":
		return transfer.FLOW_LINUX_ZEROCOPY
	case "zerocopy":
		return transfer.FLOW_ZEROCOPY
	case "multiple":
		return transfer.FLOW_MULTIPLE
	case "auto":
		return transfer.FLOW_AUTO
	default:
		return -1
	}
}

func forciblyCloseTCP(conn *net.TCPConn) {
	conn.SetLinger(0) // let Close send RST to forcibly close the connection
	conn.Close()      // forcibly close
}
