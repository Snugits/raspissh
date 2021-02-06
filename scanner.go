package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/mdlayher/arp"
	"net"
	"strings"
	"sync"
	"time"
	//"github.com/mdlayher/arp"
)

const SSH_PORT = 22
const SSH_CHECK_THRESHOLD_TIMEOUT = 200
var MAC_RASPBERRIES = map[string]bool{"b8:27:eb": true, "dc:a6:32": true, "e4:5f:01": true}

type localInfo struct {
	ip *net.IPNet
	iface net.Interface
}

type MACCheckingError struct {
	errors map[string]error
}

func (ers *MACCheckingError) Error() string {
	var result []string
	for ip, er := range ers.errors {
		result = append(result, fmt.Sprintf("%s: %s", ip, er))
	}
	return strings.Join(result, "\n")
}

func (ers *MACCheckingError) add(ip string, err error)  {
	ers.errors[ip] = err
}

func (ers *MACCheckingError) len() int {
	return len(ers.errors)
}

func getLocalIPAddress() (*localInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		addresses, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, address := range addresses {
			// check the address type and if it is not a loopback the display it
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return &localInfo{ipnet, iface}, nil
				}
			}
		}
	}
	return nil, errors.New("the local IP was not found")
}

func isSSHOpened(checkIp net.IP) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", checkIp.String(), SSH_PORT), time.Millisecond * SSH_CHECK_THRESHOLD_TIMEOUT)
	if err != nil {
		return false
	}
	_ = conn.Close()

	return true
}

func getSSHIPs(localIP *net.IPNet) ([]net.IP, error) {
	//convert IPNet struct mask and address to uint32
	//network is BigEndian
	mask := binary.BigEndian.Uint32(localIP.Mask)
	start := binary.BigEndian.Uint32(localIP.IP.To4())

	// find the final address
	finish := (start & mask) | (mask ^ 0xffffffff)

	// loop through addresses as uint32
	var ips []net.IP
	wg := &sync.WaitGroup{}
	for i := start; i <= finish; i++ {
		wg.Add(1)
		go func(i uint32) {
			defer wg.Done()
			// convert back to net.IP
			ip := make(net.IP, 4)
			binary.BigEndian.PutUint32(ip, i)
			if isSSHOpened(ip) {
				ips = append(ips, ip)
			}
		}(i)
	}

	wg.Wait()
	return ips, nil
}

func GetRaspberryPiIP() ([]net.IP, error) {
	localInfo, err := getLocalIPAddress()
	if err != nil {
		return nil, err
	}

	ips, err := getSSHIPs(localInfo.ip)

	if err != nil {
		return nil, err
	}

	raspberryIPs, err := filterRaspberryIPS(ips, &localInfo.iface)

	return raspberryIPs, nil
}

func filterRaspberryIPS(ips []net.IP, iface *net.Interface) ([]net.IP, error) {
	//@todo HERE NEED CHECK FOR SUDO!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	connection, err := arp.Dial(iface)
	if err != nil {
		return nil, err
	}
	defer connection.Close()

	var result []net.IP
	var errs *MACCheckingError
	wg := &sync.WaitGroup{}
	for _, ip := range ips {
		wg.Add(1)
		go func(ip net.IP) {
			defer wg.Done()
			mac, err := connection.Resolve(ip)
			if err != nil {
				if errs == nil {
					errs = &MACCheckingError{}
				}
				errs.add(ip.String(), err)
				return
			}
			vendorsMAC := strings.ToLower(mac.String()[:8])
			if MAC_RASPBERRIES[vendorsMAC] {
				result = append(result, ip)
			}
		}(ip)
	}
	wg.Wait()
	return result, errs
}