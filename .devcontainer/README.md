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
