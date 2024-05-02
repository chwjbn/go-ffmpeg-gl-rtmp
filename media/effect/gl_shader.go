package effect

import (
	"github.com/go-gl/gl/v3.3-core/gl"
	"os"
)

type GlShader struct {
	handle uint32
}

func NewGlShader(src string, sType uint32) (*GlShader, error) {

	handle := gl.CreateShader(sType)
	glSrc, freeFn := gl.Strs(src + "\x00")
	defer freeFn()
	gl.ShaderSource(handle, 1, glSrc, nil)
	gl.CompileShader(handle)
	err := getGlError(handle, gl.COMPILE_STATUS, gl.GetShaderiv, gl.GetShaderInfoLog,
		"SHADER::COMPILE_FAILURE::")
	if err != nil {
		return nil, err
	}

	return &GlShader{handle: handle}, nil
}

func NewGlShaderFromFile(file string, sType uint32) (*GlShader, error) {

	src, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	handle := gl.CreateShader(sType)
	glSrc, freeFn := gl.Strs(string(src) + "\x00")
	defer freeFn()

	gl.ShaderSource(handle, 1, glSrc, nil)
	gl.CompileShader(handle)
	err = getGlError(handle, gl.COMPILE_STATUS, gl.GetShaderiv, gl.GetShaderInfoLog,
		"SHADER::COMPILE_FAILURE::"+file)

	if err != nil {
		return nil, err
	}
	return &GlShader{handle: handle}, nil
}

func (shader *GlShader) Delete() {
	gl.DeleteShader(shader.handle)
}
