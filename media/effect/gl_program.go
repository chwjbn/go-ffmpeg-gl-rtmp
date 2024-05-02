package effect

import "github.com/go-gl/gl/v3.3-core/gl"

type GlProgram struct {
	handle  uint32
	shaders []*GlShader
}

func (prog *GlProgram) Delete() {
	for _, shader := range prog.shaders {
		shader.Delete()
	}
	gl.DeleteProgram(prog.handle)
}

func (prog *GlProgram) Attach(shaders ...*GlShader) {
	for _, shader := range shaders {
		gl.AttachShader(prog.handle, shader.handle)
		prog.shaders = append(prog.shaders, shader)
	}
}

func (prog *GlProgram) Use() {
	gl.UseProgram(prog.handle)
}

func (prog *GlProgram) Link() error {
	gl.LinkProgram(prog.handle)
	return getGlError(prog.handle, gl.LINK_STATUS, gl.GetProgramiv, gl.GetProgramInfoLog,
		"GlProgram::LINKING_FAILURE")
}

func (prog *GlProgram) GetUniformLocation(name string) int32 {
	return gl.GetUniformLocation(prog.handle, gl.Str(name+"\x00"))
}

func NewGlProgram(shaders ...*GlShader) (*GlProgram, error) {

	prog := &GlProgram{handle: gl.CreateProgram()}
	prog.Attach(shaders...)

	if err := prog.Link(); err != nil {
		return nil, err
	}

	return prog, nil
}
