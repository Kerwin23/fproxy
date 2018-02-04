package builder

import (
	"container/list"
	"encoding/json"
	store "fproxy/store"
	"github.com/golang/glog"
	"strconv"
	"strings"
)

type IPSection struct {
	Start    string
	End      string
	ProxyNum int
}

type IPSectionManager struct {
	Redis    *store.RedisManager
	Distance int
}

func (i *IPSectionManager) MergeStoreSections() {
	secSize, err := i.Redis.Len(KEY_SCAN_TASK)
	if err != nil {
		glog.Errorln("redis command len error", err)
		return
	}
	if secSize <= 100 {
		return
	}
	newkey := KEY_SCAN_TASK + ":check"
	i.Redis.Rename(KEY_SCAN_TASK, newkey)
	values, err := i.Redis.Lrange(newkey, 0, secSize)
	if err != nil {
		glog.Errorln("lrange ip section error: ", err)
		i.Redis.RpopLpush(newkey, KEY_SCAN_TASK)
	}
	ipSections, err := i.doMerge(values)
	if err != nil {
		glog.Errorln("merge ip section error: ", err)
		i.Redis.RpopLpush(newkey, KEY_SCAN_TASK)
	}
	i.pushIPSections(ipSections)
	i.Redis.Del(newkey)
}

func (i *IPSectionManager) pushIPSections(ipSections []*IPSection) {
	if ipSections == nil {
		return
	}
	for _, ipSection := range ipSections {
		bVal, err := json.Marshal(ipSection)
		if err != nil {
			glog.Errorln("ip section manage marshal ip section error[", ipSection.Start, ", ", ipSection.End, ", ", ipSection.ProxyNum, "]: ", err)
			return
		}
		i.Redis.Rpush(KEY_SCAN_TASK, string(bVal))
	}
}

func (i *IPSectionManager) doMerge(sections [][]byte) ([]*IPSection, error) {
	size := len(sections)
	ipSections := make([]*IPSection, size)
	for i, v := range sections {
		ipSection := &IPSection{}
		err := json.Unmarshal(v, ipSection)
		if err != nil {
			return nil, err
		}
		ipSections[i] = ipSection
	}
	return i.MergeSections(ipSections)
}

func (i *IPSectionManager) MergeSections(ipSections []*IPSection) ([]*IPSection, error) {
	secList1 := ipSectionArrayToList(ipSections)
	for {
		oldLength := secList1.Len()
		secList2 := list.New()
		e1 := secList1.Front()
		secList1.Remove(e1)
		for {
			e2 := secList1.Front()
			secList1.Remove(e2)
			sec1 := e1.Value.(*IPSection)
			sec2 := e2.Value.(*IPSection)
			sec1StartIPParts := strings.Split(sec1.Start, ".")
			sec1EndIPParts := strings.Split(sec1.End, ".")
			c1StartPart, _ := strconv.Atoi(sec1StartIPParts[2])
			c1EndPart, _ := strconv.Atoi(sec1StartIPParts[2])
			sec2StartIPParts := strings.Split(sec2.Start, ".")
			sec2EndIPParts := strings.Split(sec2.End, ".")
			if sec1StartIPParts[0] != sec2StartIPParts[0] || sec1StartIPParts[1] != sec1StartIPParts[1] {
				secList2.PushBack(e2)
				continue
			}
			c2StartPart, _ := strconv.Atoi(sec2StartIPParts[2])
			c2EndPart, _ := strconv.Atoi(sec2EndIPParts[2])
			if isSectionIntersect(c1StartPart, c1EndPart, c2StartPart, c2EndPart) || isNearSection(c1StartPart, c1EndPart, c2StartPart, c2EndPart, i.Distance) {
				if c2StartPart < c1StartPart {
					sec1StartIPParts[2] = strconv.Itoa(c2StartPart)
				}
				if c2EndPart > c1EndPart {
					c2EndPartStr := strconv.Itoa(c2EndPart)
					sec1EndIPParts[2] = c2EndPartStr
				}
				startSec := strings.Join(sec1StartIPParts, ".")
				endSec := strings.Join(sec1EndIPParts, ".")
				newIPSec := &IPSection{Start: startSec, End: endSec, ProxyNum: -1}
				e1.Value = newIPSec
			}
			if secList1.Len() == 0 {
				break
			}
		}
		secList2.PushBack(e1)
		secList1 = secList2
		if oldLength == secList2.Len() {
			break
		}
	}
	destSecs := ipSectionListToArray(secList1)
	return destSecs, nil
}

func ipSectionArrayToList(ipSections []*IPSection) *list.List {
	secList := list.New()
	for _, sec := range ipSections {
		secList.PushBack(sec)
	}
	return secList
}

func ipSectionListToArray(secList *list.List) []*IPSection {
	length := secList.Len()
	secArray := make([]*IPSection, length)
	e := secList.Front()
	for i := 0; i < length; i++ {
		secArray[i] = e.Value.(*IPSection)
		e = e.Next()
	}
	return secArray
}

func isSectionIntersect(x1, x2, y1, y2 int) bool {
	b1 := x1 <= y1 && y1 <= x2
	b2 := y1 <= x1 && x1 <= y2
	return b1 || b2
}

func isNearSection(x1, x2, y1, y2, distance int) bool {
	dis1 := x2 - y1
	if x2 < y1 {
		dis1 = y1 - x2
	}
	b1 := dis1 <= distance
	dis2 := y2 - x1
	if y2 < x1 {
		dis2 = x1 - y2
	}
	b2 := dis2 <= distance
	return b1 || b2
}
