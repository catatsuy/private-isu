package main

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

// 比較用: 修正前の実装（opensslプロセスを起動する版）を再現したもの。
// app.go内には残っていない、このベンチマーク専用のコード。
func oldDigest(ctx context.Context, src string) string {
	arg := "'" + strings.Replace(src, "'", "'\\''", -1) + "'"
	out, err := exec.CommandContext(ctx, "/bin/bash", "-c", `printf "%s" `+arg+` | openssl dgst -sha512 | sed 's/^.*= //'`).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(string(out), "\n")
}

func BenchmarkDigestOld(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		oldDigest(ctx, "password:salt")
	}
}

func BenchmarkDigestNew(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		digest(ctx, "password:salt")
	}
}
