package ipc

import (
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

const SocketName = "circadia/daemon.sock"

func GetSocketPath() (string, error) {
	return xdg.RuntimeFile(SocketName)
}

func SendSignal(msg string) error {
	path, err := GetSocketPath()
	if err != nil {
		return err
	}

	conn, err := net.Dial("unix", path)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(msg + "\n"))
	return err
}

func StartListener(callback func(string)) error {
	path, err := GetSocketPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	os.Remove(path)

	listener, err := net.Listen("unix", path)
	if err != nil {
		return err
	}

	go func() {
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("IPC Accept error: %v", err)
				continue
			}
			go handleConn(conn, callback)
		}
	}()

	return nil
}

func handleConn(conn net.Conn, callback func(string)) {
	defer conn.Close()
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}
	msg := string(buf[:n])
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}
	callback(msg)
}
