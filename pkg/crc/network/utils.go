package network

import (
	"fmt"
	"github.com/code-ready/crc/pkg/crc/logging"
	"net"
	"net/url"
	"runtime"

	"github.com/code-ready/crc/pkg/crc/errors"
	"github.com/code-ready/crc/pkg/crc/ssh"
)

func executeCommandOrExit(sshRunner *ssh.SSHRunner, command string, errorMessage string) string {
	result, err := sshRunner.Run(command)

	if err != nil {
		errors.ExitWithMessage(1, fmt.Sprintf("%s: %s", errorMessage, err.Error()))
	}
	return result
}

// NetworkContains returns true if the IP address belongs to the network given
func NetworkContains(network string, ip string) bool {
	_, ipnet, _ := net.ParseCIDR(network)
	address := net.ParseIP(ip)
	return ipnet.Contains(address)
}

// HostIPs returns the IP addresses assigned to the host
func HostIPs() []string {
	ips := []string{}

	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}

			ips = append(ips, addr.String())
		}
	}

	return ips
}

func DetermineHostIP(instanceIP string) (string, error) {
	for _, hostaddr := range HostIPs() {

		if NetworkContains(hostaddr, instanceIP) {
			hostip, _, _ := net.ParseCIDR(hostaddr)
			// This step is not working with Windows + VirtualBox as of now
			// This test is required for CIFS mount-folder case.
			// Details: https://github.com/minishift/minishift/issues/2561
			/*if IsIPReachable(driver, hostip.String(), false) {
				return hostip.String(), nil
			}*/
			return hostip.String(), nil
		}
	}

	return "", errors.New("unknown error occurred")
}

func UriStringForDisplay(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	if u.User != nil {
		return fmt.Sprintf("%s://%s:xxx@%s", u.Scheme, u.User.Username(), u.Host), nil
	}
	return uri, nil
}

func CheckCRCLocalDNSReachableFromHost(clusterName, domainName, appsDomainName string) error {
	ip, err := net.LookupIP(fmt.Sprintf("api.%s.%s", clusterName, domainName))
	if err != nil {
		return err
	}
	logging.Debugf("api.%s.%s resolved to %s", clusterName, domainName, ip)

	if runtime.GOOS != "darwin" {
		/* This check will fail with !CGO_ENABLED builds on darwin as
		 * in this case, /etc/resolver/ will not be used, so we won't
		 * have wildcard DNS for our domains
		 */
		ip, err = net.LookupIP(fmt.Sprintf("foo.%s", appsDomainName))
		if err != nil {
			return err
		}
		logging.Debugf("foo.%s resolved to %s", appsDomainName, ip)
	}
	return nil
}
