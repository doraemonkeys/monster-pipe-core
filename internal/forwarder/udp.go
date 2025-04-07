package forwarder

import (
	"net"
	"time"

	syncgmap "github.com/doraemonkeys/sync-gmap"
)

type UdpListener struct {
	listener *net.UDPConn
	// dataChMap map[string]chan []byte
	dataChMap *syncgmap.SyncMap[string, chan []byte]
}

type UdpConn struct {
	*net.UDPConn
	addr    *net.UDPAddr
	data    []byte
	dataCh  <-chan []byte
	onClose func()
}

func (u *UdpConn) Read(b []byte) (int, error) {
	if len(u.data) != 0 {
		n := copy(b, u.data)
		u.data = u.data[n:]
		return n, nil
	}
	data := <-u.dataCh
	n := copy(b, data)
	u.data = data[n:]
	return n, nil
}

func (u *UdpConn) Write(b []byte) (int, error) {
	return u.UDPConn.WriteToUDP(b, u.addr)
}

func (u *UdpConn) Close() error {
	u.onClose()
	return nil
}

func (u *UdpConn) LocalAddr() net.Addr {
	return u.UDPConn.LocalAddr()
}

func (u *UdpConn) RemoteAddr() net.Addr {
	return u.addr
}

func (u *UdpListener) Accept() (net.Conn, error) {
	const channelSize = 100
	for {
		buf := make([]byte, 2048)
		n, addr, err := u.listener.ReadFromUDP(buf)
		if err != nil {
			return nil, err
		}
		dataCh, ok := u.dataChMap.Load(addr.String())
		if ok {
			// dataCh <- buf[:n]
			if len(dataCh) < channelSize {
				dataCh <- buf[:n]
			} else {
				go func() {
					select {
					case dataCh <- buf[:n]:
					case <-time.After(time.Second):
					}
				}()
			}
			continue
		}
		dataCh = make(chan []byte, channelSize)
		u.dataChMap.Store(addr.String(), dataCh)
		return &UdpConn{
			UDPConn: u.listener,
			addr:    addr,
			data:    buf[:n],
			dataCh:  dataCh,
			onClose: func() {
				u.dataChMap.Delete(addr.String())
			},
		}, nil
	}
}

func (u *UdpListener) Close() error {
	return u.listener.Close()
}

func (u *UdpListener) Addr() net.Addr {
	return u.listener.LocalAddr()
}
