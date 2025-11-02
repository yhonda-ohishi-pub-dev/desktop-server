---
description: 変更を日本語でコミットし、パッチバージョンをアップしてpush
---

以下の手順でリリースを実行してください：

1. 現在のgit statusを確認
2. 変更されたファイルをすべてステージング
3. 変更内容を分析して、日本語でわかりやすいコミットメッセージを作成
   - コミットメッセージには必ず以下を含める：
   ```
   🤖 Generated with [Claude Code](https://claude.com/claude-code)

   Co-Authored-By: Claude <noreply@anthropic.com>
   ```
4. 最新のタグを取得してパッチバージョンをインクリメント
   - 例: v1.11.0 → v1.11.1
5. 新しいバージョンタグを作成
6. masterブランチとタグをpush
7. バイナリをビルド
8. GitHubリリースを作成してバイナリをアップロード

すべての手順を自動的に実行してください。
