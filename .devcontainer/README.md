# Dev Container

このプロジェクトはDev Containersに対応しています。

## 使用方法

1. VS Codeで「Dev Containers」拡張機能をインストール
2. コマンドパレット（`Cmd+Shift+P`）から「Dev Containers: Reopen in Container」を実行
3. コンテナがビルドされ、開発環境が自動的にセットアップされます

## 含まれる環境

- **Go 1.25.5**: プロジェクトで使用しているGoバージョン
- **golangci-lint**: リンター
- **Task**: タスクランナー（自動インストール）
- **GitHub CLI**: GitHub操作用
- **zsh + Oh My Zsh**: 快適なシェル環境

## VS Code拡張機能

以下の拡張機能が自動的にインストールされます：

- Go (golang.go)
- GitHub Copilot
- GitLens
- Task
- EditorConfig

## タスクコマンド

```bash
task test              # テスト実行
task test:coverage     # カバレッジ付きテスト
task lint              # リント実行
task lint:fix          # リント自動修正
task build             # ビルド
task fmt               # フォーマット
task check             # 全チェック実行
task ci                # CI相当のチェック
```

## 1Password SSH/GPG連携

このdevcontainerは、ホストマシンの1Password SSH Agentと連携して、SSH認証とGPG署名をサポートします。

### 必要な準備（ホストマシン側）

1. **1Password 8以降**をインストール
2. **1Password SSH Agent**を有効化
   - 1Password設定 → Developer → SSH Agentを有効化
3. SSH keyを1Passwordに保存
   - 既存のSSH keyをインポート、または1Password内で新規作成

### 動作確認

コンテナ起動後、以下のコマンドでSSH agentが正しく動作していることを確認できます：

```bash
# SSH keyの一覧表示
ssh-add -l

# SSH agent接続確認
echo $SSH_AUTH_SOCK
# 出力例: /tmp/1password-agent.sock
```

### トラブルシューティング

#### SSH agentに接続できない場合

1. ホストマシンで1Password SSH Agentが有効になっていることを確認
2. 1Passwordアプリが起動していることを確認
3. コンテナを再起動（Dev Containers: Rebuild Container）

#### Git commitでGPG署名できない場合

ホストマシンの`.gitconfig`で以下の設定を確認：

```ini
[user]
    signingkey = <your-ssh-key-from-1password>
[gpg]
    format = ssh
[commit]
    gpgsign = true
```

### .envrcファイル

プロジェクトルートの`.envrc`ファイルで、追加の環境変数を管理できます。
direnvが自動的にロードされます。
