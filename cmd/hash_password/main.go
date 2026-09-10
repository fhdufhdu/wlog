package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/alexedwards/argon2id"
	"golang.org/x/term"
)

func main() {
	fmt.Fprint(os.Stderr, "비밀번호: ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		panic(err)
	}
	hash, err := argon2id.CreateHash(string(password), argon2id.DefaultParams)
	if err != nil {
		panic(err)
	}
	fmt.Println(hash)
}
