package utils

import (
	"encoding/binary"
	"net"
	"strconv"
	"strings"

	"github.com/chinaboard/coral/utils/data"

	"github.com/juju/errors"

	log "github.com/sirupsen/logrus"
)

func ShouldDirect(ip string) bool {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("error judging ip should direct: %s", ip)
		}
	}()
	if strings.HasPrefix(ip, "0.0.0.0") || strings.HasPrefix(ip, "59.82.31.112") {
		return false
	}
	_, isPrivate := HostIsIP(ip)
	if isPrivate {
		return true
	}
	ipLong, err := Ip2long(ip)
	if err != nil {
		return false
	}
	if ipLong == 0 {
		return true
	}
	firstByte := ipLong >> 24
	if CNIPDataRange[firstByte].end == 0 {
		return false
	}
	ipIndex := SearchRange(CNIPDataRange[firstByte].start, CNIPDataRange[firstByte].end, func(i int) bool {
		return data.CNIPDataStart[i] > ipLong
	})
	ipIndex--
	return ipLong <= data.CNIPDataStart[ipIndex]+(uint32)(data.CNIPDataNum[ipIndex])
}

func HostIsIP(host string) (isIP, isPrivate bool) {
	part := strings.Split(host, ".")
	if len(part) != 4 {
		return false, false
	}
	for _, i := range part {
		if len(i) == 0 || len(i) > 3 {
			return false, false
		}
		n, err := strconv.Atoi(i)
		if err != nil || n < 0 || n > 255 {
			return false, false
		}
	}
	if part[0] == "127" || part[0] == "10" || (part[0] == "192" && part[1] == "168") {
		return true, true
	}
	if part[0] == "172" {
		n, _ := strconv.Atoi(part[1])
		if 16 <= n && n <= 31 {
			return true, true
		}
	}
	return true, false
}

func Ip2long(ipstr string) (uint32, error) {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return 0, errors.New("Invalid IP")
	}
	ip = ip.To4()
	if ip == nil {
		return 0, errors.New("Not IPv4")
	}
	return binary.BigEndian.Uint32(ip), nil
}

func SearchRange(start, end int, f func(int) bool) int {
	i, j := start, end+1
	for i < j {
		h := i + (j-i)/2 // avoid overflow when computing h
		// i ≤ h < j
		if !f(h) {
			i = h + 1 // preserves f(i-1) == false
		} else {
			j = h // preserves f(j) == true
		}
	}
	return i
}

var CNIPDataRange [256]struct {
	start int
	end   int
}

func init() {
	n := len(data.CNIPDataStart)
	var curr uint32 = 0
	var preFirstByte uint32 = 0
	for i := 0; i < n; i++ {
		firstByte := data.CNIPDataStart[i] >> 24
		if curr != firstByte {
			curr = firstByte
			if preFirstByte != 0 {
				CNIPDataRange[preFirstByte].end = i - 1
			}
			CNIPDataRange[firstByte].start = i
			preFirstByte = firstByte
		}
	}
	CNIPDataRange[preFirstByte].end = n - 1
}
