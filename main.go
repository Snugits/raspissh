package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
)

const SSH_PORT = 22

func getLocalIPAddress() *net.IPNet {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet
			}
		}
	}
	return nil
}

func getIPsRange(ipNet *net.IPNet) []net.IP {
	ipv4Net := getLocalIPAddress()
	//convert IPNet struct mask and address to uint32
	//network is BigEndian
	mask := binary.BigEndian.Uint32(ipv4Net.Mask)
	start := binary.BigEndian.Uint32(ipv4Net.IP)

	// find the final address
	finish := (start & mask) | (mask ^ 0xffffffff)

	// loop through addresses as uint32
	var ips []net.IP
	for i := start; i <= finish; i++ {
		// convert back to net.IP
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, i)
		ips = append(ips, ip)
	}

	return ips
}

func main() {
	ip, err := net.ParseMAC("192.168.100.12")

	if err != nil {
		fmt.Println(ip)
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", "192.168.100.12", SSH_PORT))
	fmt.Println(conn)
	//
	//if err == nil {
	//	_ = conn.Close()
	//}
	//addr, err := net.LookupAddr("192.168.100.12")
	//fmt.Println(addr, err)

}
