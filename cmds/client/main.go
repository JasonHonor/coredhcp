package main

import (
	"flag"
	"log"
	"net"
	"time"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/client4"
	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/dhcpv6/client6"
	"github.com/insomniacslk/dhcp/netboot"
)

var (
	ver     = flag.Int("v", 4, "IP version to use")
	ifname  = flag.String("i", "wlp4s0", "Interface name")
	dryrun  = flag.Bool("dryrun", true, "Do not change network configuration")
	debug   = flag.Bool("d", false, "Print debug output")
	retries = flag.Int("r", 3, "Number of retries before giving up")
	noIfup  = flag.Bool("noifup", false, "If set, don't wait for the interface to be up")
)

func dhclient6(ifname string, attempts int, verbose bool) (*netboot.BootConf, error) {
	if attempts < 1 {
		attempts = 1
	}
	llAddr, err := dhcpv6.GetLinkLocalAddr(ifname)
	if err != nil {
		return nil, err
	}
	laddr := net.UDPAddr{
		IP:   llAddr,
		Port: dhcpv6.DefaultClientPort,
		Zone: ifname,
	}
	raddr := net.UDPAddr{
		IP:   dhcpv6.AllDHCPRelayAgentsAndServers,
		Port: dhcpv6.DefaultServerPort,
		Zone: ifname,
	}
	c := client6.NewClient()
	c.LocalAddr = &laddr
	c.RemoteAddr = &raddr
	var conv []dhcpv6.DHCPv6
	for attempt := 0; attempt < attempts; attempt++ {
		log.Printf("Attempt %d of %d", attempt+1, attempts)
		conv, err = c.Exchange(ifname, dhcpv6.WithNetboot)
		if err != nil && attempt < attempts {
			log.Printf("Error: %v", err)
			continue
		}
		break
	}
	if verbose {
		for _, m := range conv {
			log.Print(m.Summary())
		}
	}
	if err != nil {
		return nil, err
	}
	// extract the network configuration
	netconf, err := netboot.ConversationToNetconf(conv)
	return netconf, err
}

func dhclient4(ifname string, attempts int, verbose bool) (*netboot.BootConf, error) {
	if attempts < 1 {
		attempts = 1
	}
	client := client4.NewClient()
	var (
		conv []*dhcpv4.DHCPv4
		err  error
	)
	for attempt := 0; attempt < attempts; attempt++ {
		log.Printf("Attempt %d of %d", attempt+1, attempts)
		conv, err = client.Exchange(ifname)
		if err != nil && attempt < attempts {
			log.Printf("Error: %v", err)
			continue
		}
		break
	}
	if verbose {
		for _, m := range conv {
			log.Print(m.Summary())
		}
	}
	if err != nil {
		return nil, err
	}
	// extract the network configuration
	netconf, err := netboot.ConversationToNetconfv4(conv)
	return netconf, err
}

func main() {
	flag.Parse()

	var (
		err     error
		netconf *netboot.BootConf
	)
	// bring interface up
	if !*noIfup {
		_, err = netboot.IfUp(*ifname, 5*time.Second)
		if err != nil {
			log.Fatal(err)
		}
	}
	if *ver == 6 {
		netconf, err = dhclient6(*ifname, *retries+1, *debug)
	} else {
		netconf, err = dhclient4(*ifname, *retries+1, *debug)
	}
	if err != nil {
		log.Fatal(err)
	}
	// configure the interface
	log.Printf("Setting network configuration:")
	log.Printf("%+v", netconf)
	/*	if err := netboot.ConfigureInterface(*ifname, netconf); err != nil {
		log.Fatal(err)
	}*/
}
