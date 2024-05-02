package grtmp

import (
	"fmt"
	"github.com/chwjbn/live-hub/glog"
	"github.com/chwjbn/live-hub/media/gconfig"
	"github.com/chwjbn/livego/av"
	"github.com/chwjbn/livego/protocol/rtmp"
	"github.com/chwjbn/livego/protocol/rtmp/core"
	"net"
)

type RtmpServer struct {
	mServerAddr   string
	mServerListen net.Listener
	mStream       *rtmp.RtmpStream
	mPktChan      chan av.Packet
}

func NewRtmpServer(addr string) (*RtmpServer, error) {

	xThis := new(RtmpServer)
	xThis.mServerAddr = addr

	xErr := xThis.init()

	if xErr != nil {
		return xThis, xErr
	}

	return xThis, xErr

}

func (this *RtmpServer) init() error {

	var xErr error

	var netErr error
	this.mServerListen, netErr = net.Listen("tcp", this.mServerAddr)

	if netErr != nil {
		xErr = fmt.Errorf("RTMP server listen on [%v] error:%v", this.mServerAddr, netErr.Error())
		return xErr
	}

	this.mStream = rtmp.NewRtmpStream()

	this.mPktChan = make(chan av.Packet, 4096)

	return xErr

}

func (this *RtmpServer) PushAvPacket(pkt av.Packet) error {

	var xErr error

	if pkt.IsVideo {
		glog.InfoF("$$$$$PushAvPacket Video TimeMils=[%v]", pkt.TimeStamp)
	}

	if pkt.IsAudio {
		glog.InfoF("#####PushAvPacket Audio TimeMils=[%v]", pkt.TimeStamp)
	}

	this.mPktChan <- pkt

	return xErr

}

func (this *RtmpServer) RunLoop() error {

	var xErr error

	for {
		netConn, netErr := this.mServerListen.Accept()
		if netErr != nil {
			continue
		}

		conn := core.NewConn(netConn, 64)

		go func() {

			glog.InfoF("client from=[%v] begin", conn.RemoteAddr().String())

			rtmpErr := this.handleConn(conn)
			if rtmpErr != nil {
				glog.Error(rtmpErr.Error())
			}

		}()

	}

	return xErr
}

func (this *RtmpServer) handleConn(conn *core.Conn) error {

	var xErr error

	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	rtmpErr := conn.HandshakeServer()
	if rtmpErr != nil {
		xErr = fmt.Errorf("client from=[%v] HandshakeServer error:[%v]", conn.RemoteAddr().String(), rtmpErr.Error())
		return xErr
	}

	connServer := core.NewConnServer(conn)

	rtmpErr = connServer.ReadMsg()
	if rtmpErr != nil {
		xErr = fmt.Errorf("client from=[%v] ReadMsg error:[%v]", conn.RemoteAddr().String(), rtmpErr.Error())
		return xErr
	}

	liveApp, liveChannel, liveUrl := connServer.GetInfo()
	glog.InfoF("client from=[%v] liveApp=[%v] liveChannel=[%v],liveUrl=[%v]", conn.RemoteAddr().String(), liveApp, liveChannel, liveUrl)

	//只需要订阅者
	if connServer.IsPublisher() {
		xErr = fmt.Errorf("client from=[%v] is publisher,not support", conn.RemoteAddr().String())
		return xErr
	}

	xLiveTaskMeta := gconfig.GetTaskMeta(liveChannel)
	if len(xLiveTaskMeta.VideoStreamType) < 1 {
		xErr = fmt.Errorf("client from=[%v] liveChannel=[%v] no task meta", conn.RemoteAddr().String(), liveChannel)
		return xErr
	}

	writer := rtmp.NewVirWriter(connServer)
	defer func() {
		closeErr := fmt.Errorf("client is closed")
		writer.Close(closeErr)
	}()

	for {

		if !writer.Alive() {
			break
		}

		var outPkt av.Packet
		var outPktOk bool
		outPkt, outPktOk = <-this.mPktChan

		if outPktOk {
			rtmpErr = writer.WriteBlock(&outPkt)
			if rtmpErr != nil {
				glog.ErrorF("writer.Write error:%v", rtmpErr.Error())
				break
			}
		}

		writer.SetPreTime()

	}

	return xErr

}
