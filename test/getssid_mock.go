package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	ssid := os.Getenv("MOCK_SSID")
	if ssid == "" {
		ssid = "TEST_SSID"
	}

	args := strings.Join(os.Args, " ")

	switch {
	case strings.Contains(args, "iwgetid -r"):
		// Linux向け：単純なSSID出力
		fmt.Println(ssid)

	case strings.Contains(args, "ipconfig getsummary en0"):
		// macOS向け：辞書っぽい出力
		fmt.Printf(`<dictionary> {
  InterfaceType : WiFi
  LinkStatusActive : TRUE
  SSID : %s
  Security : WPA3_SAE
}
`, ssid)

	case strings.Contains(args, "netsh wlan show interfaces"):
		// Windows向け：日本語インターフェース詳細
		fmt.Printf(`システムに 1 インターフェイスがあります:

    名前                   : Wi-Fi
    説明                   : HogeFuga Wireless LAN Adapter
    GUID                   : xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxxx
    物理アドレス           : xx:xx:xx:xx:xx:xx
    状態                   : 接続されました
    SSID                   : %s
    BSSID                  : xx:xx:xx:xx:xx:xx
    ネットワークの種類     : インフラストラクチャ
    無線の種類             : 802.11n
    認証                   : WPA2-パーソナル
    暗号                   : CCMP
    接続モード             : プロファイル
    チャネル               : 1
    受信速度 (Mbps)        : 300
    送信速度 (Mbps)        : 200
    シグナル               : 100%%
    プロファイル           : test_profile

    ホストされたネットワークの状態: 未開始
`, ssid)

	default:
		// 何も渡されなかったらTEST_SSID出す
		fmt.Println(ssid)
	}
}
