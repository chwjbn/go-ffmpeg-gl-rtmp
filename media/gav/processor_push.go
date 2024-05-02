package gav

import (
	"fmt"
	"github.com/chwjbn/live-hub/media/flv"
	"github.com/chwjbn/livego/av"
	"github.com/moonfdd/ffmpeg-go/ffcommon"
	"github.com/moonfdd/ffmpeg-go/libavcodec"
	"github.com/moonfdd/ffmpeg-go/libavformat"
	"github.com/moonfdd/ffmpeg-go/libavutil"
	"unsafe"
)

func (p *AvProcessor) pushVideoFrame(fmtCtx *libavformat.AVFormatContext, srcCodecMeta *AvCodecMeta, srcFrame *libavformat.AVFrame) error {

	var xErr error

	var dstPacket *libavformat.AVPacket

	defer func() {

		if dstPacket != nil {
			dstPacket.AvFreePacket()
		}

	}()

	dstPacket = libavcodec.AvPacketAlloc()

	timeBaseIn := fmtCtx.GetStream(ffcommon.FUnsignedInt(srcCodecMeta.StreamIndex)).TimeBase
	timeBaseOut := p.mOutVideoCodecCtx.TimeBase

	if p.mOutVideoCodecCtx.AvcodecSendFrame(srcFrame) >= 0 {

		for p.mOutVideoCodecCtx.AvcodecReceivePacket(dstPacket) >= 0 {

			dstPacket.StreamIndex = ffcommon.FUint(p.mOutVideoStreamIndex)

			dstPacket.Pts = libavutil.AvRescaleQRnd(dstPacket.Pts, timeBaseIn, timeBaseOut, libavutil.AV_ROUND_NEAR_INF|libavutil.AV_ROUND_PASS_MINMAX)
			dstPacket.Dts = libavutil.AvRescaleQRnd(dstPacket.Dts, timeBaseIn, timeBaseOut, libavutil.AV_ROUND_NEAR_INF|libavutil.AV_ROUND_PASS_MINMAX)
			dstPacket.Duration = libavutil.AvRescaleQ(dstPacket.Duration, timeBaseIn, timeBaseOut)

			dstPacket.Pos = -1

			if dstPacket.Pts < 0 {
				dstPacket.Pts = 0
			}

			//推送
			var xOutPkt av.Packet
			xOutPkt.IsVideo = true
			xOutPkt.IsAudio = false
			xOutPkt.IsMetadata = false
			xOutPkt.StreamID = uint32(srcCodecMeta.StreamIndex)
			xOutPkt.TimeStamp = uint32(dstPacket.Pts)

			//时间戳单位MS
			curPosMs := float64(dstPacket.Pts) * libavutil.AvQ2d(timeBaseOut) * 1000
			xOutPkt.TimeStamp = uint32(curPosMs)

			xPtr := uintptr(unsafe.Pointer(dstPacket.Data))

			var i uint32
			for i = 0; i < dstPacket.Size; i++ {
				xOutPkt.Data = append(xOutPkt.Data, *(*byte)(unsafe.Pointer(xPtr)))
				xPtr++
			}

			bIsKeyFrame := srcFrame.KeyFrame > 0

			//转成flv videotag
			xOutPkt.Data = flv.WriteVideoTag(xOutPkt.Data, bIsKeyFrame, flv.FLV_AVC, 0, false)

			p.mOutRtmpServer.PushAvPacket(xOutPkt)

			//p.mOutFmtCtx.AvInterleavedWriteFrame(dstPacket)

			dstPacket.AvPacketUnref()

		}

	}

	return xErr

}

func (p *AvProcessor) pushAudioFrame(fmtCtx *libavformat.AVFormatContext, srcCodecMeta *AvCodecMeta, srcFrame *libavformat.AVFrame) error {

	var xErr error

	var dstPacket *libavformat.AVPacket

	defer func() {
		if dstPacket != nil {
			dstPacket.AvFreePacket()
		}
	}()

	nRes := p.mOutAudioCodecCtx.AvcodecSendFrame(srcFrame)
	if nRes < 0 {
		xErr = fmt.Errorf("pushAudioFrame AvcodecSendFrame return error res=[%v]", nRes)
		return xErr
	}

	dstPacket = libavcodec.AvPacketAlloc()

	timeBaseIn := fmtCtx.GetStream(ffcommon.FUnsignedInt(srcCodecMeta.StreamIndex)).TimeBase
	timeBaseOut := p.mOutAudioCodecCtx.TimeBase

	for {

		nRes = p.mOutAudioCodecCtx.AvcodecReceivePacket(dstPacket)
		if nRes < 0 {
			break
		}

		dstPacket.StreamIndex = ffcommon.FUint(p.mOutAudioStreamIndex)
		dstPacket.Pts = libavutil.AvRescaleQRnd(dstPacket.Pts, timeBaseIn, timeBaseOut, libavutil.AV_ROUND_NEAR_INF|libavutil.AV_ROUND_PASS_MINMAX)
		dstPacket.Dts = libavutil.AvRescaleQRnd(dstPacket.Dts, timeBaseIn, timeBaseOut, libavutil.AV_ROUND_NEAR_INF|libavutil.AV_ROUND_PASS_MINMAX)
		dstPacket.Duration = libavutil.AvRescaleQ(dstPacket.Duration, timeBaseIn, timeBaseOut)
		dstPacket.Pos = -1

		if dstPacket.Pts < 0 {
			dstPacket.Pts = 0
		}

		//推送
		var xOutPkt av.Packet
		xOutPkt.IsVideo = false
		xOutPkt.IsAudio = true
		xOutPkt.IsMetadata = false
		xOutPkt.StreamID = uint32(srcCodecMeta.StreamIndex)
		xOutPkt.TimeStamp = uint32(dstPacket.Pts)

		//时间戳单位MS
		curPosMs := float64(dstPacket.Pts) * libavutil.AvQ2d(timeBaseOut) * 1000
		xOutPkt.TimeStamp = uint32(curPosMs)

		xPtr := uintptr(unsafe.Pointer(dstPacket.Data))

		var i uint32
		for i = 0; i < dstPacket.Size; i++ {
			xOutPkt.Data = append(xOutPkt.Data, *(*byte)(unsafe.Pointer(xPtr)))
			xPtr++
		}

		//转成flv audiotag
		xOutPkt.Data = flv.WriteAudioTag(xOutPkt.Data, flv.FLV_AAC, int(p.mOutAudioCodecCtx.SampleRate), int(p.mOutAudioCodecCtx.Channels), false)

		p.mOutRtmpServer.PushAvPacket(xOutPkt)

		//p.mOutFmtCtx.AvInterleavedWriteFrame(dstPacket)

		dstPacket.AvPacketUnref()

	}

	return xErr

}
