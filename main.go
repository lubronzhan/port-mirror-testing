package main

import (
	"fmt"
	"os"

	"github.com/vishvananda/netlink"

	"golang.org/x/sys/unix"
)

func main() {
	args := os.Args[1:]

	if len(args) != 2 {
		fmt.Println("Incorrect number of args")
		os.Exit(1)
	}
	// name of nic which traffic will be mirrored from
	fromNICName := args[0]
	fromNIC, err := netlink.LinkByName(fromNICName)
	if err != nil {
		fmt.Printf("Failed to find nic %s: %v", fromNICName, err)
		os.Exit(1)
	}
	fromNICID := fromNIC.Attrs().Index

	// name of nic which traffic will be mirrored to
	toNICName := args[1]
	toNIC, err := netlink.LinkByName(toNICName)
	if err != nil {
		fmt.Printf("Failed to find nic %s: %v", toNICName, err)
		os.Exit(1)
	}
	toNICID := toNIC.Attrs().Index

	fmt.Printf("traffic will be mirrored from interface %s to interface %s\n", fromNICName, toNICName)
	fmt.Printf("interface %s has index %d\n", fromNICName, fromNICID)
	fmt.Printf("interface %s has index %d\n", toNICName, toNICID)

	fmt.Printf("step 1: tc qdisc add dev %s ingress\n", fromNICName)
	qdisc1 := &netlink.Ingress{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: fromNICID,
			Parent:    netlink.HANDLE_INGRESS,
		},
	}

	if err := netlink.QdiscAdd(qdisc1); err != nil {
		fmt.Printf("Failed to add qdisc for index %d : %v", fromNICID, err)
		os.Exit(1)
	}

	fmt.Printf("step 2: tc filter add dev %s parent ffff: protocol ip u32 match u8 0 0 action mirred egress mirror dev %s\n", fromNICName, toNICName)
	// add a filter to mirror traffic from index1 to index2
	filter1 := &netlink.U32{
		FilterAttrs: netlink.FilterAttrs{
			LinkIndex: fromNICID,
			Parent:    netlink.MakeHandle(0xffff, 0),
			Protocol:  unix.ETH_P_ALL,
		},
		Actions: []netlink.Action{
			&netlink.MirredAction{
				ActionAttrs: netlink.ActionAttrs{
					Action: netlink.TC_ACT_PIPE,
				},
				MirredAction: netlink.TCA_EGRESS_MIRROR,
				Ifindex:      toNICID,
			},
		},
	}

	if err := netlink.FilterAdd(filter1); err != nil {
		fmt.Printf("Failed to add filter for index %d: %v", fromNICID, err)
		os.Exit(1)
	}

	fmt.Printf("step 3: tc qdisc add dev %s ingress\n", fromNICName)
	qdiscTemp := netlink.NewPrio(netlink.QdiscAttrs{
		LinkIndex: fromNICID,
		Parent:    netlink.HANDLE_ROOT,
	})

	if err := netlink.QdiscReplace(qdiscTemp); err != nil {
		fmt.Printf("Failed to replace qdisc with prio type qdisc: %v", err)
		os.Exit(1)
	}

	// get id through tc qdisc show dev vnet1
	qs, err := netlink.QdiscList(&netlink.Ifb{LinkAttrs: netlink.LinkAttrs{Index: fromNICID}})
	if err != nil {
		fmt.Printf("Failed to list qdisc for interface index %d: %v", fromNICID, err)
		os.Exit(1)
	}
	var qdiscID uint32
	for _, q := range qs {
		if q.Type() == "prio" {
			qdiscID = q.Attrs().Handle
			break
		}
	}
	if qdiscID == 0 {
		fmt.Printf("no qdisc under index %d is prio type: %v", fromNICID, err)
		os.Exit(1)
	}

	fmt.Printf("step 4: tc filter add dev %s parent %d: protocol ip u32 match u8 0 0 action mirred egress mirror dev %s\n", fromNICName, qdiscID, toNICName)

	filter2 := &netlink.U32{
		FilterAttrs: netlink.FilterAttrs{
			LinkIndex: fromNICID,
			Parent:    netlink.MakeHandle(uint16(qdiscID), 0),
			Protocol:  unix.ETH_P_ALL,
		},
		Actions: []netlink.Action{
			&netlink.MirredAction{
				ActionAttrs: netlink.ActionAttrs{
					Action: netlink.TC_ACT_PIPE,
				},
				MirredAction: netlink.TCA_EGRESS_MIRROR,
				Ifindex:      toNICID,
			},
		},
	}

	if err := netlink.FilterAdd(filter2); err != nil {
		fmt.Printf("Failed to add filter for index %d: %v", fromNICID, err)
		os.Exit(1)
	}

	fmt.Printf("That's it! Now try to tcpdump on your interface %s, and try send request to process through interface %s's ip\n", toNICName, fromNICName)
}
