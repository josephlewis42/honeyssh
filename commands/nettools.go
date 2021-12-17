package commands

import (
	"fmt"
	"strings"

	"josephlewis.net/osshit/core/vos"
)

var (
	// The commands here have been modified to be roughly consistent with respect
	// to IP and MAC addresses.
	ifconfigText = strings.TrimSpace(`
lo: flags=73<UP,LOOPBACK,RUNNING>  mtu 65536
        inet 127.0.0.1  netmask 255.0.0.0
        inet6 ::1  prefixlen 128  scopeid 0x10<host>
        loop  txqueuelen 1000  (Local Loopback)
        RX packets 687175  bytes 67648738 (67.6 MB)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 687175  bytes 67648738 (67.6 MB)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0

ens4: flags=4163<BROADCAST,MULTICAST,UP,LOWER_UP>  mtu 1500
        inet 10.128.0.2   netmask 255.255.255.0  broadcast 10.128.0.2
        inet6 fe80::4001:aff:fe80:2  prefixlen 64  scopeid 0x20<link>
        ether 42:01:0a:80:00:02  txqueuelen 1000  (Ethernet)
        RX packets 44923709  bytes 57490779806 (57.4 GB)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 12923339  bytes 2665088356 (2.6 GB)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0
`)

	ipAddress = strings.TrimSpace(`
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host
       valid_lft forever preferred_lft forever
2: ens4: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1460 qdisc mq state UP group default qlen 1000
    link/ether 42:01:0a:80:00:02 brd ff:ff:ff:ff:ff:ff
    inet 10.128.0.2/32 brd 10.128.0.2 scope global dynamic ens4
       valid_lft 86099sec preferred_lft 86099sec
    inet6 fe80::4001:aff:fe80:2/64 scope link
       valid_lft forever preferred_lft forever
`)

	ipLink = strings.TrimSpace(`
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN mode DEFAULT group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
2: ens4: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1460 qdisc mq state UP mode DEFAULT group default qlen 1000
    link/ether 42:01:0a:80:00:02 brd ff:ff:ff:ff:ff:ff
`)

	ipRoute = strings.TrimSpace(`
default via 10.128.0.1 dev ens4
10.128.0.1 dev ens4 scope link
`)

	ipRule = strings.TrimSpace(`
0:      from all lookup local
32766:  from all lookup main
32767:  from all lookup default
`)

	ipAddrlabel = strings.TrimSpace(`
prefix ::1/128 label 0
prefix ::/96 label 3
prefix ::ffff:0.0.0.0/96 label 4
prefix 2001::/32 label 6
prefix 2001:10::/28 label 7
prefix 3ffe::/16 label 12
prefix 2002::/16 label 2
prefix fec0::/10 label 11
prefix fc00::/7 label 5
prefix ::/0 label 1
`)

	ipNtable = strings.TrimSpace(`
inet arp_cache
    thresh1 128 thresh2 512 thresh3 1024 gc_int 30000
    refcnt 1 reachable 31176 base_reachable 30000 retrans 1000
    gc_stale 60000 delay_probe 5000 queue 101
    app_probes 0 ucast_probes 3 mcast_probes 3
    anycast_delay 1000 proxy_delay 800 proxy_queue 64 locktime 1000

inet arp_cache
    dev ens4
    refcnt 2 reachable 41140 base_reachable 30000 retrans 1000
    gc_stale 60000 delay_probe 5000 queue 101
    app_probes 0 ucast_probes 3 mcast_probes 3
    anycast_delay 1000 proxy_delay 800 proxy_queue 64 locktime 1000

inet arp_cache
    dev lo
    refcnt 2 reachable 35048 base_reachable 30000 retrans 1000
    gc_stale 60000 delay_probe 5000 queue 101
    app_probes 0 ucast_probes 3 mcast_probes 3
    anycast_delay 1000 proxy_delay 800 proxy_queue 64 locktime 1000

inet6 ndisc_cache
    thresh1 128 thresh2 512 thresh3 1024 gc_int 30000
    refcnt 1 reachable 35552 base_reachable 30000 retrans 1000
    gc_stale 60000 delay_probe 5000 queue 101
    app_probes 0 ucast_probes 3 mcast_probes 3
    anycast_delay 1000 proxy_delay 800 proxy_queue 64 locktime 0

inet6 ndisc_cache
    dev ens4
    refcnt 4 reachable 39848 base_reachable 30000 retrans 1000
    gc_stale 60000 delay_probe 5000 queue 101
    app_probes 0 ucast_probes 3 mcast_probes 3
    anycast_delay 1000 proxy_delay 800 proxy_queue 64 locktime 0

inet6 ndisc_cache
    dev lo
    refcnt 2 reachable 23036 base_reachable 30000 retrans 1000
    gc_stale 60000 delay_probe 5000 queue 101
    app_probes 0 ucast_probes 3 mcast_probes 3
    anycast_delay 1000 proxy_delay 800 proxy_queue 64 locktime 0
`)

	ipNetconf = strings.TrimSpace(`
    inet lo forwarding off rp_filter off mc_forwarding off proxy_neigh off ignore_routes_with_linkdown off
    inet ens4 forwarding off rp_filter strict mc_forwarding off proxy_neigh off ignore_routes_with_linkdown off
    inet all forwarding off rp_filter strict mc_forwarding off proxy_neigh off ignore_routes_with_linkdown off
    inet default forwarding off rp_filter strict mc_forwarding off proxy_neigh off ignore_routes_with_linkdown off
    inet6 lo forwarding off mc_forwarding off proxy_neigh off ignore_routes_with_linkdown off
    inet6 ens4 forwarding off mc_forwarding off proxy_neigh off ignore_routes_with_linkdown off
    inet6 all forwarding off mc_forwarding off proxy_neigh off ignore_routes_with_linkdown off
    inet6 default forwarding off mc_forwarding off proxy_neigh off ignore_routes_with_linkdown off
`)

	ipTunnel = ""
)

// Ifconfig implements the ifconfig command.
func Ifconfig(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "ifconfig [OPTION...]",
		Short: "configure a network interface",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		fmt.Fprintln(virtOS.Stdout(), ifconfigText)
		return 0
	})
}

// Ip implements the ip command (newer replacemnet for ifconfig)
func Ip(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "ip [ OPTIONS ] (link | address | addrlabel | route | rule | ntable | tunnel)",
		Short: "configure routing, devices, interfaces, and tunnels",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		opt := ""
		if args := cmd.Flags().Args(); len(args) > 0 {
			opt = args[0]
		}

		toDisplay := ""
		switch opt {
		case "link":
			toDisplay = ipLink
		case "route":
			toDisplay = ipRoute
		case "rule":
			toDisplay = ipRule
		case "tunnel":
			toDisplay = ""
		case "addrlabel":
			toDisplay = ipAddrlabel
		case "ntable":
			toDisplay = ipNtable
		case "address", "":
			fallthrough
		default:
			toDisplay = ipAddress
		}

		fmt.Fprintln(virtOS.Stdout(), toDisplay)
		return 0
	})
}

var _ vos.ProcessFunc = Ifconfig

func init() {
	addSbinCmd("ifconfig", Ifconfig)
	addSbinCmd("ip", Ip)
}
