package prompt

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNormalize(t *testing.T) {
	in := " \x1b[31m認証\x1b[0m\x1b7\x1b(B\n\t処理\x00\x07   を修正 "
	if got, want := Normalize(in), "認証 処理 を修正"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWidthAndTruncateJapanese(t *testing.T) {
	if got := Width("abc日本"); got != 7 {
		t.Fatalf("width = %d", got)
	}
	if got, want := Truncate("日本語です", 7), "日本語…"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTruncateDoesNotSplitGrapheme(t *testing.T) {
	in := "A👨‍👩‍👧‍👦B"
	got := Truncate(in, 3)
	if !utf8.ValidString(got) || !strings.HasSuffix(got, "…") {
		t.Fatalf("invalid truncation %q", got)
	}
	if strings.Contains(got, "\u200d…") {
		t.Fatalf("split grapheme: %q", got)
	}
}

func TestTruncateTinyWidth(t *testing.T) {
	if got := Truncate("abc", 1); got != "…" {
		t.Fatalf("got %q", got)
	}
	if got := Truncate("abc", 0); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestDisplayPrefix(t *testing.T) {
	if got, want := DisplayPrefix("\x1b[31mTask:\x1b[0m\n"), "Task: "; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := DisplayPrefix("\x00\n"); got != "" {
		t.Fatalf("got %q", got)
	}
}
