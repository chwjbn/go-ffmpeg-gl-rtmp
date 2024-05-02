package effect

import (
	"fmt"
	"github.com/go-gl/gl/v3.3-core/gl"
	"image"
)

type GlTexture struct {
	handle  uint32
	target  uint32
	texUnit uint32
}

func NewGlTexture() (*GlTexture, error) {

	var xErr error

	var handle uint32
	gl.GenTextures(1, &handle)

	target := uint32(gl.TEXTURE_2D)

	texture := GlTexture{
		handle: handle,
		target: target,
	}

	texture.Bind(gl.TEXTURE0)
	defer texture.UnBind()

	gl.TexParameteri(texture.target, gl.TEXTURE_WRAP_R, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(texture.target, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(texture.target, gl.TEXTURE_MIN_FILTER, gl.LINEAR) // minification filter
	gl.TexParameteri(texture.target, gl.TEXTURE_MAG_FILTER, gl.LINEAR) // magnification filter

	return &texture, xErr
}

func (tex *GlTexture) SetImage(img *image.RGBA) error {

	var xErr error

	tex.Bind(gl.TEXTURE0)
	defer tex.UnBind()

	if img.Stride != img.Rect.Size().X*4 {
		xErr = fmt.Errorf("unsupported stride, only 32-bit colors supported")
		return xErr
	}

	width := int32(img.Rect.Size().X)
	height := int32(img.Rect.Size().Y)
	dataPtr := gl.Ptr(img.Pix)

	internalFmt := int32(gl.SRGB_ALPHA)
	format := uint32(gl.RGBA)
	pixType := uint32(gl.UNSIGNED_BYTE)

	gl.TexImage2D(tex.target, 0, internalFmt, width, height, 0, format, pixType, dataPtr)
	gl.GenerateMipmap(tex.target)

	return xErr

}

func (tex *GlTexture) SetUniform(uniformLoc int32) error {
	if tex.texUnit == 0 {
		return fmt.Errorf("texture not bound")
	}
	gl.Uniform1i(uniformLoc, int32(tex.texUnit-gl.TEXTURE0))
	return nil
}

func (tex *GlTexture) Bind(texUnit uint32) {
	gl.ActiveTexture(texUnit)
	gl.BindTexture(tex.target, tex.handle)
	tex.texUnit = texUnit
}

func (tex *GlTexture) UnBind() {
	tex.texUnit = 0
	gl.BindTexture(tex.target, 0)
}
