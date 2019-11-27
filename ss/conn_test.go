package ss

import (
	"bytes"
	"io"
	"net"
	"testing"
)

func mustNewCipher(method string) *Cipher {
	const testPassword = "password"
	cipher, err := NewCipher(method, testPassword)
	if err != nil {
		panic(err)
	}
	return cipher
}

// transcript 副本
type transcriptConn struct {
	net.Conn
	ReadTranscript []byte
}

func (conn *transcriptConn) Read(p []byte) (int, error) {
	n, err := conn.Conn.Read(p)
	conn.ReadTranscript = append(conn.ReadTranscript, p[:n]...)
	return n, err
}

func connIVs(method string) (clientIV, serverIV []byte, err error) {
	// 模拟一个网络连接
	clientConn, serverConn := net.Pipe()
	// 底层数据交换， 副本记录
	clientTranscriptConn := &transcriptConn{Conn: clientConn}
	serverTranscriptConn := &transcriptConn{Conn: serverConn}

	// shasowsocks 级别的 连接
	clientSSConn := NewConn(clientTranscriptConn, mustNewCipher(method))
	serverSSConn := NewConn(serverTranscriptConn, mustNewCipher(method))

	clientToServerDate := []byte("client to server data")
	serverToClientData := []byte("server to client data")

	go func() { // 模拟 ss server
		defer serverConn.Close()
		buf := make([]byte, len(clientToServerDate))
		_, err := io.ReadFull(serverSSConn, buf)
		if err != nil {
			return
		}

		_, err = serverSSConn.Write(serverToClientData)
		if err != nil {
			return
		}
	}()

	// 模拟 ss client
	defer clientSSConn.Close()
	_, err = clientSSConn.Write(clientToServerDate)
	if err != nil {
		return
	}
	buf := make([]byte, len(serverToClientData))
	_, err = io.ReadFull(clientSSConn, buf)
	if err != nil {
		return
	}

	// 模拟完成， 取出 IV
	clientIV = serverTranscriptConn.ReadTranscript[:clientSSConn.Cipher.info.ivLen]
	serverIV = clientTranscriptConn.ReadTranscript[:serverSSConn.Cipher.info.ivLen]
	return
}

func Test_IndependentIVs(t *testing.T) {
	// server client 互换 iv , 他们不该相等
	for method := range cipherMethod {
		clientIV, serverIV, err := connIVs(method)
		if err != nil {
			t.Errorf("%s connection err: %s\n", method, err)
			continue
		}
		if bytes.Equal(clientIV, serverIV) {
			t.Errorf("%s equal client and server IVs\n", method)
			continue
		}
		if debug {
			t.Logf("%s: clientIV %x serverIV %x\n", method, clientIV, serverIV)
		}
	}
}
