package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/vishvananda/netlink"

	"golang.org/x/sys/unix"
)

func main() {
	args := os.Args[1:]

	if len(args) != 2 {
		fmt.Println("Incorrect number of args")
		os.Exit(1)
	}

	// index of nic which will be mirrored from
	index1, _ := strconv.Atoi(args[0])
	// index of nic which will be mirrored to
	index2, _ := strconv.Atoi(args[1])

	fmt.Printf("network index1 : %d\n", index1)
	fmt.Printf("network index2 : %d\n", index2)

	fmt.Println("step 1: tc qdisc add dev vnet1 ingress")
	qdisc1 := &netlink.Ingress{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: index1,
			Parent:    netlink.HANDLE_INGRESS,
		},
	}

	if err := netlink.QdiscAdd(qdisc1); err != nil {
		fmt.Printf("Failed to add qdisc for index %d : %s", index1, err)
		os.Exit(1)
	}

	fmt.Println("step 2: tc filter add dev vnet1 parent ffff: protocol ip u32 match u8 0 0 action mirred egress mirror dev vnet0")
	// add a filter to mirror traffic from index1 to index2
	filter1 := &netlink.U32{
		FilterAttrs: netlink.FilterAttrs{
			LinkIndex: index1,
			Parent:    netlink.MakeHandle(0xffff, 0),
			Protocol:  unix.ETH_P_ALL,
		},
		Actions: []netlink.Action{
			&netlink.MirredAction{
				ActionAttrs: netlink.ActionAttrs{
					Action: netlink.TC_ACT_PIPE,
				},
				MirredAction: netlink.TCA_EGRESS_MIRROR,
				Ifindex:      index2,
			},
		},
	}

	if err := netlink.FilterAdd(filter1); err != nil {
		fmt.Printf("Failed to add filter for index %d: %v", index1, err)
		os.Exit(1)
	}

	fmt.Println("step 3: tc qdisc add dev vnet1 ingress")
	qdiscTemp := netlink.NewPrio(netlink.QdiscAttrs{
		LinkIndex: index1,
		Parent:    netlink.HANDLE_ROOT,
	})

	if err := netlink.QdiscReplace(qdiscTemp); err != nil {
		fmt.Printf("Failed to replace qdisc with prio type qdisc: %v", err)
		os.Exit(1)
	}

	fmt.Println("step 4: tc filter add dev vnet1 parent 8002: protocol ip u32 match u8 0 0 action mirred egress mirror dev vnet0")
	// get id through tc qdisc show dev vnet1
	qs, err := netlink.QdiscList(&netlink.Ifb{LinkAttrs: netlink.LinkAttrs{Index: index1}})
	if err != nil {
		fmt.Printf("Failed to list qdisc for interface index %d: %v", index1, err)
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
		fmt.Printf("no qdisc under index %d is prio type: %v", index1, err)
		os.Exit(1)
	}

	filter2 := &netlink.U32{
		FilterAttrs: netlink.FilterAttrs{
			LinkIndex: index1,
			Parent:    netlink.MakeHandle(uint16(qdiscID), 0),
			Protocol:  unix.ETH_P_ALL,
		},
		Actions: []netlink.Action{
			&netlink.MirredAction{
				ActionAttrs: netlink.ActionAttrs{
					Action: netlink.TC_ACT_PIPE,
				},
				MirredAction: netlink.TCA_EGRESS_MIRROR,
				Ifindex:      index2,
			},
		},
	}

	if err := netlink.FilterAdd(filter2); err != nil {
		fmt.Printf("Failed to add filter for index %d: %v", index1, err)
		os.Exit(1)
	}

	fmt.Printf("That's it! Now try to tcpdump on your interface %d, and try send request to process through interface %d's ip\n", index2, index1)
}
