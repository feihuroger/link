package packnet

import (
	"net"
	"sync"
	"sync/atomic"
)

// Session.
type Session struct {
	id uint64

	// About network
	conn   net.Conn
	writer PacketWriter
	reader PacketReader

	// About send and receive
	sendChan       chan Response
	sendPacketChan chan []byte
	requestHandler RequestHandler

	// About session close
	closeChan     chan int
	closeWait     *sync.WaitGroup
	closeFlag     int32
	closeCallback func(*Session)

	// Put your session state here.
	State interface{}
}

func NewSession(id uint64, conn net.Conn, writer PacketWriter, reader PacketReader, sendChanSize uint) *Session {
	return &Session{
		id:             id,
		conn:           conn,
		writer:         writer,
		reader:         reader,
		sendChan:       make(chan Response, sendChanSize),
		sendPacketChan: make(chan []byte, sendChanSize),
		closeChan:      make(chan int),
		closeWait:      new(sync.WaitGroup),
		closeFlag:      -1,
	}
}

func (session *Session) Start() {
	if atomic.CompareAndSwapInt32(&session.closeFlag, -1, 0) {
		go session.writeLoop()
		go session.readLoop()
	} else {
		panic(SessionDuplicateStartError)
	}
}

// Loop and wait incoming requests.
func (session *Session) readLoop() {
	session.closeWait.Add(1)
	defer func() {
		session.closeWait.Done()
		session.Close()
	}()

	var (
		packet []byte
		err    error
	)

	for {
		packet, err = session.reader.ReadPacket(session.conn, packet)
		if err != nil {
			break
		}
		session.requestHandler.Handle(session, packet)
	}
}

// Loop and transport responses.
func (session *Session) writeLoop() {
	session.closeWait.Add(1)
	defer func() {
		session.closeWait.Done()
		session.Close()
	}()

	var packet []byte
L:
	for {
		select {
		case response := <-session.sendChan:
			size := response.RecommendPacketSize()

			packet = session.writer.BeginPacket(size, packet)
			packet = response.AppendToPacket(packet)
			packet = session.writer.EndPacket(packet)

			if err := session.writer.WritePacket(session.conn, packet); err != nil {
				break L
			}
		case packet := <-session.sendPacketChan:
			if err := session.writer.WritePacket(session.conn, packet); err != nil {
				break L
			}
		case <-session.closeChan:
			break L
		}
	}
}

// Get session id.
func (session *Session) Id() uint64 {
	return session.id
}

// Get local address.
func (session *Session) RawConn() net.Conn {
	return session.conn
}

// Set session close callback.
func (session *Session) SetCloseCallback(callback func(*Session)) {
	session.closeCallback = callback
}

// Set request handler.
func (session *Session) SetRequestHandler(requestHandler RequestHandler) {
	session.requestHandler = requestHandler
}

func (session *Session) SetRequestHandlerFunc(callback func(*Session, []byte)) {
	session.requestHandler = requestHandlerFunc{callback}
}

// Close session and remove it from api server.
func (session *Session) Close() {
	if atomic.CompareAndSwapInt32(&session.closeFlag, 0, 1) {
		// if close session without this goroutine
		// deadlock will happen when session close by itself.
		go func() {
			defer func() {
				// remove session from server
				if session.closeCallback != nil {
					session.closeCallback(session)
				}
			}()

			session.conn.Close()

			// notify write loop session closed
			close(session.closeChan)

			// wait for read loop and write lopp exit
			session.closeWait.Wait()
		}()
	}
}

// Async send a response.
func (session *Session) Send(response Response) error {
	if atomic.LoadInt32(&session.closeFlag) != 0 {
		return SendToClosedError
	}

	select {
	case session.sendChan <- response:
		return nil
	default:
		session.Close()
		return BlockingError
	}
}

// Async send a packet.
func (session *Session) SendPacket(packet []byte) error {
	if atomic.LoadInt32(&session.closeFlag) != 0 {
		return SendToClosedError
	}

	select {
	case session.sendPacketChan <- packet:
		return nil
	default:
		session.Close()
		return BlockingError
	}
}