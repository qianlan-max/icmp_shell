// qianlan_icmpsh/cmd/server/cli.go
package main

import (
	"fmt"
	"github.com/urfave/cli"
	"lichu_icmpsh/server"
	"log"
	"os"
)

const (
	MinMtu = 64
)

var (
	app = &cli.App{
		Name:  "icmp_shell-ser",
		Usage: "BY ~~qianlan~~",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "token, t", Usage: "握手Token", Value: "qianlan"},
			cli.BoolFlag{Name: "filetrans, ft", Usage: "文件接收模式 (与-m, -cm互斥)"},
			cli.StringFlag{Name: "crypto-mode, cm", Usage: "加密模式 (可选: none, xor, base64, base32, aes)", Value: "none"},
			cli.StringFlag{Name: "mode, m", Usage: "运行模式 (可选: session, beacon)", Value: "session"},
			cli.IntFlag{Name: "mtu", Usage: fmt.Sprintf("单包最大载荷 (字节, 最小: %d)", MinMtu), Value: 666},
		},
		Action: func(c *cli.Context) error {
			// --- 参数check ---
			mtu := c.Int("mtu")
			if mtu < MinMtu {
				return fmt.Errorf("错误: --mtu 的值 (%d) 过小，必须大于等于 %d", mtu, MinMtu)
			}
			token := c.String("token")
			if len(token) >= mtu {
				return fmt.Errorf("错误: --token 的长度 (%d) 不能大于或等于 --mtu 的值 (%d)", len(token), mtu)
			}

			if c.Bool("filetrans") {
				if c.IsSet("mode") || c.IsSet("crypto-mode") {
					return fmt.Errorf("错误: --filetrans 模式与 --mode、--crypto-mode 不同时使用")
				}
				fmt.Println("--- ICMP隧道服务端 (文件接收模式) ---")
				config := server.ServerConfig{
					Token:         []byte(token),
					FileTransMode: true,
				}
				s, err := server.NewServer(config)
				if err != nil {
					log.Fatalf("错误: 初始化服务端失败: %v", err)
				}
				s.ListenForFile()
				return nil
			}

			mode := c.String("mode")
			if mode != "session" && mode != "beacon" {
				return fmt.Errorf("无效的运行模式: %s, 请选择 'session' 或 'beacon'", mode)
			}
			fmt.Printf("--- ICMP隧道服务端 ---\n")
			fmt.Printf("[+] 运行模式: %s\n", mode)
			fmt.Printf("[+] 加密模式: %s\n", c.String("crypto-mode"))
			fmt.Printf("[+] 单包最大载荷 (MTU): %d 字节\n", mtu)
			fmt.Println("[+] 等待客户端连接...")

			config := server.ServerConfig{
				Token:      []byte(token),
				CryptoMode: c.String("crypto-mode"),
				Mode:       mode,
				Mtu:        mtu,
			}
			s, err := server.NewServer(config)
			if err != nil {
				log.Fatalf("错误: 初始化服务端失败: %v", err)
			}

			go s.ListenICMP()
			err = s.StartupShell()
			if err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}
)

func main() {
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
