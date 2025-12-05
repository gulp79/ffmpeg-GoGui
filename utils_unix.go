//go:build !windows

package main

import "os/exec"

func setSysProcAttr(cmd *exec.Cmd) {
	// Nessuna azione necessaria su Linux/Mac per nascondere finestre console figlie di solito
}
