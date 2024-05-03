package gav

import "C"
import (
	"fmt"
	"github.com/chwjbn/live-hub/glog"
	"github.com/chwjbn/live-hub/media/effect"
	"github.com/chwjbn/live-hub/media/gconfig"
	"github.com/chwjbn/live-hub/media/grtmp"
	"github.com/moonfdd/ffmpeg-go/ffcommon"
	"github.com/moonfdd/ffmpeg-go/libavcodec"
	"github.com/moonfdd/ffmpeg-go/libavformat"
	"github.com/moonfdd/ffmpeg-go/libavutil"
	"github.com/moonfdd/ffmpeg-go/libswresample"
	"golang.org/x/sys/windows"
	"strconv"
	"unsafe"
)

type AvProcessor struct {
	mEffectProcessor *effect.EffectProcessor
	mTaskMeta        *gconfig.TaskMeta

	mOutFmtCtx           *libavformat.AVFormatContext
	mOutVideoStreamIndex int32
	mOutAudioStreamIndex int32
	mOutVideoCodecCtx    *libavformat.AVCodecContext
	mOutAudioCodecCtx    *libavformat.AVCodecContext

	mOutRtmpServer *grtmp.RtmpServer

	mInFmtCtx         *libavformat.AVFormatContext
	mInVideoCodecMeta *AvCodecMeta
	mInAudioCodecMeta *AvCodecMeta

	mInVideoRotate int

	mAudioSwrContext *libswresample.SwrContext
	mAudioFifo       *libavutil.AVAudioFifo
}

func NewAvProcessor(effectProcessor *effect.EffectProcessor, taskMeta *gconfig.TaskMeta) (*AvProcessor, error) {

	pThis := new(AvProcessor)

	pThis.mEffectProcessor = effectProcessor
	pThis.mTaskMeta = taskMeta

	xErr := pThis.init()

	if xErr != nil {
		return nil, xErr
	}

	return pThis, nil
}

func (p *AvProcessor) init() error {

	var xErr error

	p.mOutRtmpServer, xErr = grtmp.NewRtmpServer("0.0.0.0:1949")
	if xErr != nil {
		return xErr
	}

	go func() {
		p.mOutRtmpServer.RunLoop()
	}()

	xErr = InitAvRuntime()
	if xErr != nil {
		return xErr
	}

	//初始化输入
	xErr = p.InitInputStream()
	if xErr != nil {
		return xErr
	}

	//初始化输出
	xErr = p.InitOutStream()
	if xErr != nil {
		return xErr
	}

	//分配FIFO队列
	p.mAudioFifo = libavutil.AvAudioFifoAlloc(p.mOutAudioCodecCtx.SampleFmt, p.mOutAudioCodecCtx.Channels, 1)

	p.mAudioSwrContext = libswresample.SwrAlloc()
	p.mAudioSwrContext = p.mAudioSwrContext.SwrAllocSetOpts(
		int64(p.mOutAudioCodecCtx.ChannelLayout),
		p.mOutAudioCodecCtx.SampleFmt,
		p.mOutAudioCodecCtx.SampleRate,
		int64(p.mInAudioCodecMeta.CodecCtx.ChannelLayout),
		p.mInAudioCodecMeta.CodecCtx.SampleFmt,
		p.mInAudioCodecMeta.CodecCtx.SampleRate,
		0,
		uintptr(0))

	res := p.mAudioSwrContext.SwrInit()
	if res < 0 {
		xErr = fmt.Errorf("AudioSwrContext.SwrInit faild")
		return xErr
	}

	return xErr

}

func (p *AvProcessor) InitInputStream() error {

	var xErr error

	var res ffcommon.FInt

	filePath := p.mTaskMeta.VideoStreamPath

	//打开文件流
	res = libavformat.AvformatOpenInput(&p.mInFmtCtx, filePath, nil, nil)
	if res < 0 {
		xErr = fmt.Errorf("AvformatOpenInput return error res=[%v]", res)
		return xErr
	}

	// 获取流信息
	res = p.mInFmtCtx.AvformatFindStreamInfo(nil)
	if res < 0 {
		xErr = fmt.Errorf("AvformatFindStreamInfo return error res=[%v]", res)
		return xErr
	}

	p.mInFmtCtx.AvDumpFormat(0, filePath, 0)

	var inVideoStreamIndex = -1
	var inAudioStreamIndex = -1

	for i := uint32(0); i < p.mInFmtCtx.NbStreams; i++ {
		if p.mInFmtCtx.GetStream(i).Codecpar.CodecType == libavutil.AVMEDIA_TYPE_VIDEO {
			inVideoStreamIndex = int(i)
			break
		}
	}

	for i := uint32(0); i < p.mInFmtCtx.NbStreams; i++ {
		if p.mInFmtCtx.GetStream(i).Codecpar.CodecType == libavutil.AVMEDIA_TYPE_AUDIO {
			inAudioStreamIndex = int(i)
			break
		}
	}

	if inVideoStreamIndex < 0 {
		xErr = fmt.Errorf("no video stream")
		return xErr
	}

	if inAudioStreamIndex < 0 {
		xErr = fmt.Errorf("no audio stream")
		return xErr
	}

	glog.InfoF("media file=[%v] video stream index=[%v]", filePath, inVideoStreamIndex)
	glog.InfoF("media file=[%v] audio stream index=[%v]", filePath, inAudioStreamIndex)

	var codecErr error

	p.mInVideoCodecMeta, codecErr = NewAvCodecMeta(p.mInFmtCtx.GetStream(ffcommon.FUnsignedInt(inVideoStreamIndex)))

	if codecErr != nil {
		xErr = fmt.Errorf("create video stream decoder error:[%v]", codecErr.Error())
		return xErr
	}

	p.mInAudioCodecMeta, codecErr = NewAvCodecMeta(p.mInFmtCtx.GetStream(ffcommon.FUnsignedInt(inAudioStreamIndex)))
	if codecErr != nil {
		xErr = fmt.Errorf("create audio stream decoder error:[%v]", codecErr.Error())
		return xErr
	}

	//获取视频旋转角度
	var xTagErr error
	var xTag *libavutil.AVDictionaryEntry
	xTag = p.mInFmtCtx.GetStream(ffcommon.FUnsignedInt(p.mInVideoCodecMeta.StreamIndex)).Metadata.AvDictGet("rotate", nil, 0)
	if xTag != nil {
		xTagVal := windows.BytePtrToString((*byte)(unsafe.Pointer(xTag.Value)))
		if len(xTagVal) > 0 {
			p.mInVideoRotate, xTagErr = strconv.Atoi(xTagVal)
			if xTagErr != nil {
				p.mInVideoRotate = 0
			}
		}
	}

	timeSecTotal := p.mInFmtCtx.Duration / libavutil.AV_TIME_BASE
	glog.InfoF("file path=[%v] timeSecTotal=[%v]s", filePath, timeSecTotal)

	return xErr
}

func (p *AvProcessor) initOutVideoStream() error {

	var xErr error

	stream := p.mOutFmtCtx.AvformatNewStream(nil)
	if stream == nil {
		xErr = fmt.Errorf("create output video stream faild")
		return xErr
	}
	p.mOutVideoStreamIndex = stream.Index

	if p.mInVideoRotate > 0 {
		libavutil.AvDictSet(&stream.Metadata, "rotate", fmt.Sprintf("%v", p.mInVideoRotate), 0)
	}

	stream.TimeBase.Num = 1
	stream.TimeBase.Den = 30
	stream.Codecpar.CodecId = libavcodec.AV_CODEC_ID_H264
	stream.Codecpar.CodecType = libavutil.AVMEDIA_TYPE_VIDEO
	stream.Codecpar.Format = libavutil.AV_PIX_FMT_YUV420P
	stream.Codecpar.Width = ffcommon.FInt(p.mTaskMeta.DstVideoWidth)
	stream.Codecpar.Height = ffcommon.FInt(p.mTaskMeta.DstVideoHeight)
	stream.Codecpar.BitRate = 128 * 1024
	stream.Codecpar.CodecTag = 0

	//硬件编码
	var outCodec *libavcodec.AVCodec

	if outCodec == nil {
		outCodec = libavcodec.AvcodecFindEncoderByName("h264_nvenc")
	}

	if outCodec == nil {
		outCodec = libavcodec.AvcodecFindEncoderByName("h264_qsv")
	}

	if outCodec == nil {
		outCodec = libavcodec.AvcodecFindEncoder(stream.Codecpar.CodecId)
	}

	if outCodec == nil {
		xErr = fmt.Errorf("cannot find output video encoder")
		return xErr
	}

	p.mOutVideoCodecCtx = outCodec.AvcodecAllocContext3()
	if p.mOutVideoCodecCtx == nil {
		xErr = fmt.Errorf("cannot alloc output video encoder context")
		return xErr
	}

	res := p.mOutVideoCodecCtx.AvcodecParametersToContext(stream.Codecpar)
	if res < 0 {
		xErr = fmt.Errorf("cannot apply output video encoder context")
		return xErr
	}

	p.mOutVideoCodecCtx.CodecId = stream.Codecpar.CodecId
	p.mOutVideoCodecCtx.CodecType = stream.Codecpar.CodecType
	p.mOutVideoCodecCtx.PixFmt = stream.Codecpar.Format
	p.mOutVideoCodecCtx.Width = stream.Codecpar.Width
	p.mOutVideoCodecCtx.Height = stream.Codecpar.Height
	p.mOutVideoCodecCtx.TimeBase.Num = stream.TimeBase.Num
	p.mOutVideoCodecCtx.TimeBase.Den = stream.TimeBase.Den
	p.mOutVideoCodecCtx.BitRate = stream.Codecpar.BitRate
	p.mOutVideoCodecCtx.GopSize = 25

	//参数影响画质
	if p.mOutVideoCodecCtx.CodecId == libavcodec.AV_CODEC_ID_H264 {
		p.mOutVideoCodecCtx.Qmin = 1
		p.mOutVideoCodecCtx.Qmax = 24
		p.mOutVideoCodecCtx.Qcompress = 0.5
		p.mOutVideoCodecCtx.MaxBFrames = 3
	} else if p.mOutVideoCodecCtx.CodecId == libavcodec.AV_CODEC_ID_MPEG2VIDEO {
		p.mOutVideoCodecCtx.MaxBFrames = 2
	} else if p.mOutVideoCodecCtx.CodecId == libavcodec.AV_CODEC_ID_MPEG1VIDEO {
		p.mOutVideoCodecCtx.MbDecision = 2
	}

	//打开编码器
	res = p.mOutVideoCodecCtx.AvcodecOpen2(outCodec, nil)
	if res < 0 {
		xErr = fmt.Errorf("cannot open output video encoder context")
		return xErr
	}

	return xErr

}

func (p *AvProcessor) initOutAudioStream() error {

	var xErr error

	stream := p.mOutFmtCtx.AvformatNewStream(nil)
	if stream == nil {
		xErr = fmt.Errorf("create output audio stream faild")
		return xErr
	}
	p.mOutAudioStreamIndex = stream.Index

	stream.Codecpar.CodecId = libavcodec.AV_CODEC_ID_AAC
	stream.Codecpar.CodecType = libavutil.AVMEDIA_TYPE_AUDIO

	stream.Codecpar.Format = libavutil.AV_SAMPLE_FMT_FLTP
	stream.Codecpar.SampleRate = 44100
	stream.Codecpar.ChannelLayout = libavutil.AV_CH_LAYOUT_STEREO
	stream.Codecpar.Channels = libavutil.AvGetChannelLayoutNbChannels(stream.Codecpar.ChannelLayout)
	stream.Codecpar.BitRate = 128 * 1024
	stream.Codecpar.Profile = 1
	stream.Codecpar.CodecTag = 0

	outCodec := libavcodec.AvcodecFindEncoder(stream.Codecpar.CodecId)
	if outCodec == nil {
		xErr = fmt.Errorf("cannot find output audio encoder")
		return xErr
	}

	p.mOutAudioCodecCtx = outCodec.AvcodecAllocContext3()
	if p.mOutAudioCodecCtx == nil {
		xErr = fmt.Errorf("cannot alloc output audio encoder context")
		return xErr
	}

	res := p.mOutAudioCodecCtx.AvcodecParametersToContext(stream.Codecpar)
	if res < 0 {
		xErr = fmt.Errorf("cannot apply output audio encoder context")
		return xErr
	}

	p.mOutAudioCodecCtx.CodecId = stream.Codecpar.CodecId
	p.mOutAudioCodecCtx.CodecType = stream.Codecpar.CodecType
	p.mOutAudioCodecCtx.SampleFmt = libavcodec.AVSampleFormat(stream.Codecpar.Format)
	p.mOutAudioCodecCtx.SampleRate = stream.Codecpar.SampleRate
	p.mOutAudioCodecCtx.ChannelLayout = stream.Codecpar.ChannelLayout
	p.mOutAudioCodecCtx.BitRate = stream.Codecpar.BitRate
	p.mOutAudioCodecCtx.Channels = stream.Codecpar.Channels
	p.mOutAudioCodecCtx.Profile = stream.Codecpar.Profile
	p.mOutAudioCodecCtx.ExtradataSize = stream.Codecpar.ExtradataSize
	p.mOutAudioCodecCtx.Extradata = stream.Codecpar.Extradata
	p.mOutAudioCodecCtx.CodecTag = stream.Codecpar.CodecTag
	p.mOutAudioCodecCtx.Flags |= libavcodec.AV_CODEC_FLAG_GLOBAL_HEADER

	//打开编码器
	res = p.mOutAudioCodecCtx.AvcodecOpen2(outCodec, nil)
	if res < 0 {
		xErr = fmt.Errorf("cannot open output audio encoder context")
		return xErr
	}

	return xErr

}

func (p *AvProcessor) InitOutStream() error {

	var xErr error

	//outFilePath := "rtmp://127.0.0.1:1935/live/rfBd56ti2SMtYvSgD5xAV0YU99zampta7Z7S575KLkIZ9PYk"

	outFmt := libavformat.AvGuessFormat("flv", "", "")
	if outFmt == nil {
		xErr = fmt.Errorf("invalid output format")
		return xErr
	}

	res := libavformat.AvformatAllocOutputContext2(&p.mOutFmtCtx, outFmt, "", "")
	if res < 0 {
		xErr = fmt.Errorf("alloc output format context faild")
		return xErr
	}

	xErr = p.initOutVideoStream()
	if xErr != nil {
		return xErr
	}

	xErr = p.initOutAudioStream()
	if xErr != nil {
		return xErr
	}

	/*
		if (p.mOutFmtCtx.Oformat.Flags & libavformat.AVFMT_NOFILE) == 0 {
			res = libavformat.AvioOpen(&p.mOutFmtCtx.Pb, outFilePath, libavformat.AVIO_FLAG_WRITE)
			if res < 0 {
				xErr = fmt.Errorf("open out file faild")
				return xErr
			}
		}

		res = p.mOutFmtCtx.AvformatWriteHeader(nil)
		if res < 0 {
			xErr = fmt.Errorf("write out file header faild")
			return xErr
		}*/

	p.mOutFmtCtx.AvDumpFormat(0, "", 1)

	return xErr

}

func (p *AvProcessor) closeOutStream() error {

	var xErr error

	if p.mOutAudioCodecCtx != nil {
		p.mOutAudioCodecCtx.AvcodecClose()
		p.mOutAudioCodecCtx = nil
	}

	if p.mOutVideoCodecCtx != nil {
		p.mOutVideoCodecCtx.AvcodecClose()
		p.mOutVideoCodecCtx = nil
	}

	p.mOutVideoStreamIndex = -1
	p.mOutAudioStreamIndex = -1

	if p.mOutFmtCtx != nil {
		p.mOutFmtCtx.AvformatFreeContext()
		p.mOutFmtCtx = nil
	}

	return xErr

}

func (p *AvProcessor) closeInStream() error {

	var xErr error

	if p.mInVideoCodecMeta != nil {
		p.mInVideoCodecMeta.CodecCtx.AvcodecClose()
		p.mInVideoCodecMeta = nil
	}

	if p.mInAudioCodecMeta != nil {
		p.mInAudioCodecMeta.CodecCtx.AvcodecClose()
		p.mInAudioCodecMeta = nil
	}

	if p.mInFmtCtx != nil {
		p.mInFmtCtx.AvformatFreeContext()
		p.mInFmtCtx = nil
	}

	return xErr

}

func (p *AvProcessor) RunProccess() error {

	var xErr error

	var res ffcommon.FInt

	//解码前帧包数据
	var avPkt *libavformat.AVPacket
	avPkt = libavcodec.AvPacketAlloc()

	//解码后帧数据
	var avFrame *libavformat.AVFrame
	avFrame = libavutil.AvFrameAlloc()

	defer func() {
		libavutil.AvFrameFree(&avFrame)
		avPkt.AvFreePacket()
	}()

	inVideoStream := p.mInFmtCtx.GetStream(ffcommon.FUnsignedInt(p.mInVideoCodecMeta.StreamIndex))
	inVideoFrameIndex := 0

	inAudioStream := p.mInFmtCtx.GetStream(ffcommon.FUnsignedInt(p.mInAudioCodecMeta.StreamIndex))
	inAudioFrameIndex := 0

	for {
		res = p.mInFmtCtx.AvReadFrame(avPkt)
		if res < 0 {
			break
		}

		//视频帧解码
		if avPkt.StreamIndex == uint32(p.mInVideoCodecMeta.StreamIndex) {

			if avPkt.Pts == libavutil.AV_NOPTS_VALUE {
				streamTimeBase := inVideoStream.TimeBase
				calcDuration := libavutil.AV_TIME_BASE / libavutil.AvQ2d(inVideoStream.RFrameRate)
				avPkt.Pts = int64((float64(inVideoFrameIndex) * calcDuration) / (libavutil.AvQ2d(streamTimeBase) * float64(libavutil.AV_TIME_BASE)))
				avPkt.Dts = avPkt.Pts
				avPkt.Duration = int64(calcDuration / (libavutil.AvQ2d(streamTimeBase) * float64(libavutil.AV_TIME_BASE)))
			}

			if p.mInVideoCodecMeta.CodecCtx.AvcodecSendPacket(avPkt) >= 0 {

				for {
					res = p.mInVideoCodecMeta.CodecCtx.AvcodecReceiveFrame(avFrame)
					if res < 0 {
						break
					}

					p.processVideoFrame(p.mInFmtCtx, p.mInVideoCodecMeta, avFrame)
				}

			} else {
				glog.WarnF("skip video packet=[%v]", avPkt.Pts)
			}

			inVideoFrameIndex++

		}

		//音频帧解码
		if avPkt.StreamIndex == uint32(p.mInAudioCodecMeta.StreamIndex) {

			if avPkt.Pts == libavutil.AV_NOPTS_VALUE {
				streamTimeBase := inAudioStream.TimeBase
				calcDuration := libavutil.AV_TIME_BASE / libavutil.AvQ2d(inAudioStream.RFrameRate)
				avPkt.Pts = int64((float64(inAudioFrameIndex) * calcDuration) / (libavutil.AvQ2d(streamTimeBase) * float64(libavutil.AV_TIME_BASE)))
				avPkt.Dts = avPkt.Pts
				avPkt.Duration = int64(calcDuration / (libavutil.AvQ2d(streamTimeBase) * float64(libavutil.AV_TIME_BASE)))
			}

			if p.mInAudioCodecMeta.CodecCtx.AvcodecSendPacket(avPkt) >= 0 {

				for {
					res = p.mInAudioCodecMeta.CodecCtx.AvcodecReceiveFrame(avFrame)
					if res < 0 {
						break
					}

					p.processAudioFrame(p.mInFmtCtx, p.mInAudioCodecMeta, avFrame)
				}
			} else {
				glog.WarnF("skip audio packet=[%v]", avPkt.Pts)
			}

			inAudioFrameIndex++

		}

		avPkt.AvPacketUnref()
	}

	return xErr
}
