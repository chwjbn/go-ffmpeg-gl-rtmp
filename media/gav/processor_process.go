package gav

import (
	"github.com/chwjbn/live-hub/glog"
	"github.com/moonfdd/ffmpeg-go/ffcommon"
	"github.com/moonfdd/ffmpeg-go/libavcodec"
	"github.com/moonfdd/ffmpeg-go/libavformat"
	"github.com/moonfdd/ffmpeg-go/libavutil"
	"github.com/moonfdd/ffmpeg-go/libswscale"
	"image"
	"runtime"
	"unsafe"
)

func (p *AvProcessor) processVideoFrame(fmtCtx *libavformat.AVFormatContext, srcCodecMeta *AvCodecMeta, srcFrame *libavformat.AVFrame) error {

	var xErr error

	var dstFrame *libavformat.AVFrame
	var dstFrameBuffer ffcommon.FVoidP

	var xSrc2RgbImgCtx *libswscale.SwsContext
	var xRgbImg2DstCtx *libswscale.SwsContext

	defer func() {
		if xSrc2RgbImgCtx != nil {
			xSrc2RgbImgCtx.SwsFreeContext()
		}

		if xRgbImg2DstCtx != nil {
			xRgbImg2DstCtx.SwsFreeContext()
		}

		if dstFrame != nil {
			libavutil.AvFrameFree(&dstFrame)
		}

		if dstFrameBuffer > 0 {
			libavutil.AvFree(dstFrameBuffer)
		}

	}()

	//处理目标帧
	dstFrame = libavutil.AvFrameAlloc()
	dstFrame.Width = p.mOutVideoCodecCtx.Width
	dstFrame.Height = p.mOutVideoCodecCtx.Height
	dstFrame.Format = p.mOutVideoCodecCtx.PixFmt

	dstFrameBufferSize := libavutil.AvImageGetBufferSize(dstFrame.Format, dstFrame.Width, dstFrame.Height, 1)
	dstFrameBuffer = libavutil.AvMalloc(uint64(dstFrameBufferSize))

	//填充内存
	res := libavutil.AvImageFillArrays(
		(*[4]*byte)(unsafe.Pointer(&dstFrame.Data)),
		(*[4]int32)(unsafe.Pointer(&dstFrame.Linesize)),
		(*byte)(unsafe.Pointer(dstFrameBuffer)),
		dstFrame.Format,
		dstFrame.Width,
		dstFrame.Height,
		1)

	if res < 0 {
		return xErr
	}

	xSrc2RgbImgCtx = libswscale.SwsGetContext(
		srcFrame.Width,
		srcFrame.Height,
		srcFrame.Format,
		srcFrame.Width,
		srcFrame.Height,
		libavutil.AV_PIX_FMT_RGBA,
		libswscale.SWS_BICUBIC,
		nil,
		nil,
		nil)

	if xSrc2RgbImgCtx == nil {
		return xErr
	}

	xRgbImg2DstCtx = libswscale.SwsGetContext(
		srcFrame.Width,
		srcFrame.Height,
		libavutil.AV_PIX_FMT_RGBA,
		dstFrame.Width,
		dstFrame.Height,
		dstFrame.Format,
		libswscale.SWS_BICUBIC,
		nil,
		nil,
		nil)

	if xRgbImg2DstCtx == nil {
		return xErr
	}

	rgbImg := image.NewRGBA(image.Rect(0, 0, int(srcFrame.Width), int(srcFrame.Height)))
	if rgbImg == nil {
		return xErr
	}

	rgbImgBuffPtr := uintptr(unsafe.Pointer(&rgbImg.Pix[0]))

	//转换成RGBA图片
	res = xSrc2RgbImgCtx.SwsScale(
		(**byte)(unsafe.Pointer(&srcFrame.Data)),
		(*int32)(unsafe.Pointer(&srcFrame.Linesize)),
		0,
		uint32(srcFrame.Height),
		(**byte)(unsafe.Pointer(&rgbImgBuffPtr)),
		(*int32)(unsafe.Pointer(&rgbImg.Stride)))

	if res < 0 {
		return xErr
	}

	var dstImg *image.RGBA
	dstImg, xErr = p.mEffectProcessor.ProcessAvFrameImage(rgbImg, p.mInVideoRotate)
	if xErr != nil {
		return xErr
	}

	/*
		xFile, xFileErr := os.Create(path.Join(glib.AppBaseDir(), "test.png"))
		if xFileErr == nil {
			png.Encode(xFile, dstImg)
			xFile.Close()
		}*/

	dstImgBuffPtr := uintptr(unsafe.Pointer(&dstImg.Pix[0]))

	//转换回
	res = xRgbImg2DstCtx.SwsScale(
		(**byte)(unsafe.Pointer(&dstImgBuffPtr)),
		(*int32)(unsafe.Pointer(&dstImg.Stride)),
		0,
		uint32(dstImg.Rect.Dy()),
		(**byte)(unsafe.Pointer(&dstFrame.Data)),
		(*int32)(unsafe.Pointer(&dstFrame.Linesize)))

	if res < 0 {
		return xErr
	}

	dstFrame.KeyFrame = srcFrame.KeyFrame
	dstFrame.Pts = srcFrame.Pts
	dstFrame.PktPts = srcFrame.PktPts
	dstFrame.PktDts = srcFrame.PktDts
	dstFrame.PktDuration = srcFrame.PktDuration

	if dstFrame.Pts < 0 {
		dstFrame.Pts = 0
	}

	xErr = p.pushVideoFrame(fmtCtx, srcCodecMeta, dstFrame)

	return xErr
}

func (p *AvProcessor) processAudioFrame(fmtCtx *libavformat.AVFormatContext, srcCodecMeta *AvCodecMeta, srcFrame *libavformat.AVFrame) error {

	var xErr error

	var outDataBuffer **ffcommon.FUint8T

	var dstFrame *libavformat.AVFrame
	var dstFrameBuffer ffcommon.FVoidP

	defer func() {

		if outDataBuffer != nil {

			libavutil.AvFreep(uintptr(unsafe.Pointer(outDataBuffer)))
			libavutil.AvFreep(uintptr(unsafe.Pointer(&outDataBuffer)))

			outDataBuffer = nil
		}

		if dstFrame != nil {
			libavutil.AvFrameFree(&dstFrame)
			dstFrame = nil
		}

		if dstFrameBuffer > 0 {
			libavutil.AvFree(dstFrameBuffer)
			dstFrameBuffer = 0
		}

	}()

	var res ffcommon.FInt

	outChannelLayout := p.mOutAudioCodecCtx.ChannelLayout
	outFormat := ffcommon.FInt(p.mOutAudioCodecCtx.SampleFmt)
	outChannels := p.mOutAudioCodecCtx.Channels
	outSampleRate := p.mOutAudioCodecCtx.SampleRate
	outNbSamples := p.mOutAudioCodecCtx.FrameSize

	//最大采样数
	maxOutNbSamples := int32(libavutil.AvRescaleRnd(p.mAudioSwrContext.SwrGetDelay(int64(srcFrame.SampleRate))+
		int64(srcFrame.NbSamples), int64(outSampleRate), int64(srcFrame.SampleRate), libavutil.AV_ROUND_UP))

	var outDataLineSize ffcommon.FInt
	res = libavutil.AvSamplesAllocArrayAndSamples(&outDataBuffer, &outDataLineSize, outChannels, maxOutNbSamples, libavutil.AVSampleFormat(outFormat), 1)
	if res < 0 {
		return xErr
	}

	inDataAddr := srcFrame.ExtendedData
	inDataCount := srcFrame.NbSamples

	runtime.LockOSThread()

	for {

		libavutil.AvFreep(uintptr(unsafe.Pointer(outDataBuffer)))
		res = libavutil.AvSamplesAlloc(outDataBuffer, &outDataLineSize, outChannels, maxOutNbSamples, libavutil.AVSampleFormat(outFormat), 1)
		if res < 0 {
			break
		}

		xConvertNbSamples := p.mAudioSwrContext.SwrConvert(
			outDataBuffer,
			maxOutNbSamples,
			inDataAddr,
			inDataCount)

		if xConvertNbSamples <= 0 {
			break
		}

		//溢出了
		if xConvertNbSamples > maxOutNbSamples {
			glog.WarnF("!!!!!xConvertNbSamples=[%v] maxOutNbSamples=[%v]", xConvertNbSamples, maxOutNbSamples)
			xConvertNbSamples = maxOutNbSamples
		}

		inDataAddr = (**ffcommon.FUint8T)(unsafe.Pointer(uintptr(0)))
		inDataCount = 0

		res = p.mAudioFifo.AvAudioFifoWrite((*ffcommon.FVoidP)(unsafe.Pointer(outDataBuffer)), xConvertNbSamples)
		if res <= 0 {
			break
		}
	}

	dstFrameBufferSize := libavutil.AvSamplesGetBufferSize(nil, outChannels, outNbSamples, libavutil.AVSampleFormat(outFormat), 1)
	dstFrameBuffer = libavutil.AvMalloc(ffcommon.FSizeT(dstFrameBufferSize))

	dstFrame = libavutil.AvFrameAlloc()
	dstFrame.ChannelLayout = outChannelLayout
	dstFrame.SampleRate = outSampleRate
	dstFrame.Format = outFormat
	dstFrame.Channels = outChannels
	dstFrame.NbSamples = outNbSamples

	dstFrame.KeyFrame = srcFrame.KeyFrame
	dstFrame.Pts = srcFrame.Pts
	dstFrame.PktPts = srcFrame.PktPts
	dstFrame.PktDts = srcFrame.PktDts
	dstFrame.PktDuration = srcFrame.PktDuration

	if dstFrame.Pts < 0 {
		dstFrame.Pts = 0
	}

	res = libavcodec.AvcodecFillAudioFrame(dstFrame, dstFrame.Channels, libavcodec.AVSampleFormat(dstFrame.Format),
		(*byte)(unsafe.Pointer(dstFrameBuffer)), dstFrameBufferSize, 1)

	if res < 0 {
		return xErr
	}

	for {
		leftNbSamples := p.mAudioFifo.AvAudioFifoSize()
		if leftNbSamples < outNbSamples {
			break
		}

		res = p.mAudioFifo.AvAudioFifoRead((*ffcommon.FVoidP)(unsafe.Pointer(&dstFrame.Data)), outNbSamples)
		if res < 0 {
			break
		}

		dstFrame.Pts = dstFrame.Pts + 1
		p.pushAudioFrame(fmtCtx, srcCodecMeta, dstFrame)

	}

	return xErr
}
