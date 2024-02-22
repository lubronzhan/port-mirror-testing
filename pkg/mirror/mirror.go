package mirror

import (
	"fmt"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func MirrorTrafficFromNIC(fromNICName, toNICName string) error {
	// name of nic which traffic will be mirrored from
	fromNIC, err := netlink.LinkByName(fromNICName)
	if err != nil {
		return fmt.Errorf("failed to find nic %s: %v", fromNICName, err)
	}
	fromNICID := fromNIC.Attrs().Index

	// name of nic which traffic will be mirrored to
	toNIC, err := netlink.LinkByName(toNICName)
	if err != nil {
		return fmt.Errorf("failed to find nic %s: %v", toNICName, err)
	}
	toNICID := toNIC.Attrs().Index

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
		return fmt.Errorf("failed to add qdisc for index %d : %v", fromNICID, err)
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
		return fmt.Errorf("failed to add filter for index %d: %v", fromNICID, err)
	}

	fmt.Printf("step 3: tc qdisc add dev %s ingress\n", fromNICName)
	qdiscTemp := netlink.NewPrio(netlink.QdiscAttrs{
		LinkIndex: fromNICID,
		Parent:    netlink.HANDLE_ROOT,
	})

	if err := netlink.QdiscReplace(qdiscTemp); err != nil {
		return fmt.Errorf("failed to replace qdisc with prio type qdisc: %v", err)
	}

	// get id through tc qdisc show dev vnet1
	qs, err := netlink.QdiscList(&netlink.Ifb{LinkAttrs: netlink.LinkAttrs{Index: fromNICID}})
	if err != nil {
		fmt.Printf("Failed to list qdisc for interface index %d: %v", fromNICID, err)
		return err
	}
	var qdiscID uint32
	for _, q := range qs {
		if q.Type() == "prio" {
			qdiscID = q.Attrs().Handle
			break
		}
	}
	if qdiscID == 0 {
		return fmt.Errorf("no qdisc under index %d is prio type: %v", fromNICID, err)
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
		return fmt.Errorf("failed to add filter for index %d: %v", fromNICID, err)
	}

	fmt.Printf("That's it! Now try to tcpdump on your interface %s, and try send request to process through interface %s's ip\n", toNICName, fromNICName)
	return nil
}

func CleanupQDSICFromNIC(nicName string) error {
	// name of nic which traffic will be mirrored to
	toNIC, err := netlink.LinkByName(nicName)
	if err != nil {
		return fmt.Errorf("failed to find nic %s: %v", nicName, err)
	}
	nicID := toNIC.Attrs().Index

	fmt.Printf("interface %s has index %d\n", nicName, nicID)

	fmt.Println("step 1: delete ingress qdisc")
	qdisc1 := &netlink.Ingress{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: nicID,
			Parent:    netlink.HANDLE_INGRESS,
		},
	}
	if err := netlink.QdiscDel(qdisc1); err != nil {
		return err
	}

	fmt.Println("step 2: delete root qdisc")
	qdiscRoot := netlink.NewPrio(netlink.QdiscAttrs{
		LinkIndex: nicID,
		Parent:    netlink.HANDLE_ROOT,
	})
	if err := netlink.QdiscDel(qdiscRoot); err != nil {
		return err
	}
	return nil
}
