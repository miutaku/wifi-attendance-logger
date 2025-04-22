# wifi-attendance-logger

会社のWiFi（SSID）に接続したことをトリガーに出社を記録し、SQLiteに1日1回だけ記録を残します。  
出社回数をいちいち管理しなくても、このアプリケーションを定期実行させてやれば、自動で出社日数の集計ができます。

## 特徴

- macOS / Windows 対応（Linux拡張予定）
- 特定のSSIDに接続した時だけ記録
- SQLiteで記録（ローカルファイル）
- 同日に複数回接続しても1回だけ記録
- 出社時に任意のコマンドを実行可能（Slack通知など）

## セットアップ

1. 適当な場所に `wifi-attendance-logger` バイナリを設置する

2. `config.yaml` を設置する

`wifi-attendance-logger` バイナリと同じディレクトリに配置してください。

`config.sample.yaml` がサンプルであるので、config.ymlという名前でコピーして修正してください。

また、`./wifi-attendance-logger` するたびに `config.yaml` を読み込むため、設定変更後の対応などは不要です。

3. 定期実行設定（5分おき）

- **macOS**: `launchd` を使って定期実行
- **Windows**: タスクスケジューラで `.exe` を5分おきに実行
- **Linux**: `crond` などで設定

## 出社記録をする

出社記録をするとこうなります。定期実行するようにしてください。
一日一度だけの記録が保証されているため、出社履歴のカウントで、一日あたり2回以上の出社カウントをすることがありません。

```bash
./wifi-attendance-logger
2025/04/22 14:08:49 Attendance recorded for jitaku
```

## 出社履歴の確認

WindowsならPowerAutomateを使うとか、macOSならショートカット.appを使って以下コマンドを実行するとかすると楽だと思います。

```bash
./wifi-attendance-logger -check
[今月の出社ログ]
2025-04-22 - 東京オフィス
2025-04-26 - 大阪オフィス

出社日数合計: 2 日
```

## ライセンス

MIT
