package gav

import (
	"fmt"
	"github.com/chwjbn/live-hub/glib"
	"github.com/moonfdd/ffmpeg-go/ffcommon"
	"github.com/moonfdd/ffmpeg-go/libavcodec"
	"github.com/moonfdd/ffmpeg-go/libavfilter"
	"github.com/moonfdd/ffmpeg-go/libavformat"
	"os"
	"path"
	"path/filepath"
)

func InitAvRuntime() error {

	var xErr error

	libPath := path.Join(glib.AppBaseDir(), "lib")
	if !glib.DirExists(libPath) {
		xErr = fmt.Errorf("missing ffmpeg lib files")
		return xErr
	}

	var pathErr error
	libPath, pathErr = filepath.Abs(libPath)
	if pathErr != nil {
		xErr = fmt.Errorf("invalid ffmpeg lib path")
		return xErr
	}

	envErr := os.Setenv("Path", fmt.Sprintf("%s;%s", os.Getenv("Path"), libPath))
	if envErr != nil {
		xErr = fmt.Errorf("set path error:[%s]", envErr.Error())
		return xErr
	}

	ffcommon.SetAvutilPath(path.Join(libPath, "avutil-56.dll"))
	ffcommon.SetAvcodecPath(path.Join(libPath, "avcodec-58.dll"))
	ffcommon.SetAvdevicePath(path.Join(libPath, "avdevice-58.dll"))
	ffcommon.SetAvfilterPath(path.Join(libPath, "avfilter-7"))
	ffcommon.SetAvformatPath(path.Join(libPath, "avformat-58.dll"))
	ffcommon.SetAvpostprocPath(path.Join(libPath, "postproc-55.dll"))
	ffcommon.SetAvswresamplePath(path.Join(libPath, "swresample-3.dll"))
	ffcommon.SetAvswscalePath(path.Join(libPath, "swscale-5.dll"))

	libavformat.AvRegisterAll()
	libavfilter.AvfilterRegisterAll()

	var res ffcommon.FInt
	res = libavformat.AvformatNetworkInit()
	if res < 0 {
		xErr = fmt.Errorf("AvformatNetworkInit return res=[%v]", res)
		return xErr
	}

	return xErr

}

func DeInitAvRuntime() error {

	var xErr error

	libavformat.AvformatNetworkDeinit()

	return xErr

}

type AvCodecMeta struct {
	Codec       *libavcodec.AVCodec
	CodecCtx    *libavcodec.AVCodecContext
	StreamIndex ffcommon.FInt
}

func NewAvCodecMeta(avstream *libavformat.AVStream) (*AvCodecMeta, error) {

	var xErr error
	pThis := new(AvCodecMeta)

	pThis.StreamIndex = avstream.Index

	xAvCodeId := avstream.Codecpar.CodecId

	//硬件解码
	if xAvCodeId == libavcodec.AV_CODEC_ID_H264 {

		if pThis.Codec == nil {
			pThis.Codec = libavcodec.AvcodecFindDecoderByName("h264_qsv")
		}

		if pThis.Codec == nil {
			pThis.Codec = libavcodec.AvcodecFindDecoderByName("h264_cuvid")
		}

	}

	if pThis.Codec == nil {
		pThis.Codec = libavcodec.AvcodecFindDecoder(avstream.Codecpar.CodecId)
	}

	if pThis.Codec == nil {
		xErr = fmt.Errorf("can not find stream decoder")
		return nil, xErr
	}

	pThis.CodecCtx = pThis.Codec.AvcodecAllocContext3()

	var nRes ffcommon.FInt

	nRes = pThis.CodecCtx.AvcodecParametersToContext(avstream.Codecpar)
	if nRes < 0 {
		pThis.Free()
		xErr = fmt.Errorf("AvcodecParametersToContext return error res=[%v]", nRes)
		return nil, xErr
	}

	nRes = pThis.CodecCtx.AvcodecOpen2(pThis.Codec, nil)
	if nRes < 0 {
		pThis.Free()
		xErr = fmt.Errorf("AvcodecOpen2 return error res=[%v]", nRes)
		return nil, xErr
	}

	return pThis, xErr
}

func (this *AvCodecMeta) Free() error {

	var xErr error

	if this.CodecCtx == nil {
		return xErr
	}

	this.CodecCtx.AvcodecClose()
	this.CodecCtx = nil

	return xErr

}
