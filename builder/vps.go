package builder

import (
	"encoding/json"
	"errors"
	"fproxy/store"
	"github.com/golang/glog"
	"strconv"
	"time"
)

const (
	VPS_PROXY_SET  = "proxy:vps:set"
	VPS_PROXY_DATA = "proxy:vps:data:"
	VPS_ALIVE_TIME = 290
)

type VPS struct {
	Redis *store.RedisManager
}

type VPSProxy struct {
	IP         string
	Port       int
	StartTime  int64
	LeftSecond int64
}

func (v *VPS) AddVPS(vpsName string, ip string, port int) {
	isOldVPS, err := v.Redis.Sismember(VPS_PROXY_SET, vpsName)
	if err != nil || !isOldVPS {
		v.Redis.Sadd(VPS_PROXY_SET, vpsName)
	}
	key := VPS_PROXY_DATA + vpsName
	text, err := v.Redis.Get(key)
	if err != nil {
		glog.Errorln("before add vps get from redis err[", key, "]", err)
		return
	}
	timestamp := time.Now().UnixNano()
	if text != "" {
		var oldProxy VPSProxy
		err = json.Unmarshal([]byte(text), &oldProxy)
		if err != nil {
			v.addNewVPS(vpsName, ip, port, timestamp)
		} else {
			if oldProxy.IP != ip && oldProxy.Port != port {
				v.addNewVPS(vpsName, ip, port, timestamp)
			} else {
				v.updateVPS(vpsName, ip, port, timestamp, oldProxy)
			}
		}
	} else {
		v.addNewVPS(vpsName, ip, port, timestamp)
	}
}

func (v *VPS) addNewVPS(vpsName string, ip string, port int, startTime int64) {
	key := VPS_PROXY_DATA + vpsName
	newProxy := VPSProxy{IP: ip, Port: port, StartTime: startTime, LeftSecond: VPS_ALIVE_TIME}
	btext, err := json.Marshal(newProxy)
	if err != nil {
		glog.Errorln("add new vps proxy parse to json err: ", err)
		return
	}
	v.Redis.Set(key, string(btext))
}

func (v *VPS) updateVPS(vpsName string, ip string, port int, curtime int64, oldProxy VPSProxy) {
	key := VPS_PROXY_DATA + vpsName
	startSecond := oldProxy.StartTime / 1000000000
	nowSecond := curtime / 1000000000
	leftSecond := VPS_ALIVE_TIME - (nowSecond - startSecond)
	oldProxy.LeftSecond = leftSecond
	btext, err := json.Marshal(oldProxy)
	if err != nil {
		glog.Errorln("update vps proxy parse to json err: ", err)
		return
	}
	v.Redis.Set(key, string(btext))
}

func (v *VPS) GetValidVPS() ([]string, error) {
	bTexts, err := v.Redis.Smembers(VPS_PROXY_SET)
	if err != nil {
		glog.Errorln("get valid vps from redis error: ", err)
		return nil, err
	}
	bLen := 0
	if bTexts == nil {
		return nil, errors.New("no valid vps")
	}
	bLen = len(bTexts)
	vpses := make([]string, bLen)
	nowSecond := time.Now().UnixNano() / 1000000000
	for i, bText := range bTexts {
		var proxy VPSProxy
		err = json.Unmarshal(bText, &proxy)
		if err != nil {
			return nil, errors.New("can not deserialize vps")
		}
		startSecond := proxy.StartTime / 1000000000
		useSecond := nowSecond - startSecond
		if useSecond > VPS_ALIVE_TIME {
			continue
		}
		vps := proxy.IP + ":" + strconv.Itoa(proxy.Port)
		vpses[i] = vps
	}
	return vpses, nil
}
