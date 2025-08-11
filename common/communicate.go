// lichu_icmpsh/common/common.go
package common

import (
	"errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"net"
	"time"
)

type Communicate struct {
	Src            net.IP
	Dst            net.IP
	DstHwAddr      net.HardwareAddr
	Gateway        net.IP
	Iface          *net.Interface
	PcapSendHandle *pcap.Handle
	Seq            uint16
	Mtu            int
}

func (c *Communicate) SendICMP(payload []byte, icmpId uint16, icmpType uint8) error {
	if len(payload) > c.Mtu {
		return fmt.Errorf("payload size (%d) exceeds MTU (%d), must be fragmented before calling SendICMP", len(payload), c.Mtu)
	}

	ethernetLayer := &layers.Ethernet{
		SrcMAC:       c.Iface.HardwareAddr,
		DstMAC:       c.DstHwAddr,
		EthernetType: layers.EthernetTypeIPv4,
	}
	ipLayer := &layers.IPv4{
		SrcIP:    c.Src,
		DstIP:    c.Dst,
		Version:  4,
		TTL:      64,
		Protocol: layers.IPProtocolICMPv4,
		TOS:      20,
		Id:       1000,
	}
	icmpLayer := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(icmpType, 0),
		Id:       icmpId,
		Seq:      c.Seq,
	}

	err := c.Send(ethernetLayer, ipLayer, icmpLayer, gopacket.Payload(payload))
	c.Seq++
	if err != nil {
		return err
	}

	return nil
}

func (c *Communicate) GetHwAddr() (net.HardwareAddr, error) {
	start := time.Now()
	arpDst := c.Dst
	if c.Gateway != nil {
		arpDst = c.Gateway
	}
	eth := layers.Ethernet{
		SrcMAC:       c.Iface.HardwareAddr,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}
	arp := layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   []byte(c.Iface.HardwareAddr),
		SourceProtAddress: []byte(c.Src),
		DstHwAddress:      []byte{0, 0, 0, 0, 0, 0},
		DstProtAddress:    []byte(arpDst),
	}
	if err := c.Send(&eth, &arp); err != nil {
		return nil, err
	}
	for {
		if time.Since(start) > time.Second*3 {
			return nil, errors.New("timeout getting ARP reply")
		}
		data, _, err := c.PcapSendHandle.ReadPacketData()
		if err == pcap.NextErrorTimeoutExpired {
			continue
		} else if err != nil {
			return nil, err
		}
		packet := gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.NoCopy)
		if arpLayer := packet.Layer(layers.LayerTypeARP); arpLayer != nil {
			arp := arpLayer.(*layers.ARP)
			if net.IP(arp.SourceProtAddress).Equal(net.IP(arpDst)) {
				return net.HardwareAddr(arp.SourceHwAddress), nil
			}
		}
	}
}

func (c *Communicate) Send(l ...gopacket.SerializableLayer) error {
	buffer := gopacket.NewSerializeBuffer()
	if err := gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}, l...); err != nil {
		return err
	}
	return c.PcapSendHandle.WritePacketData(buffer.Bytes())
}
