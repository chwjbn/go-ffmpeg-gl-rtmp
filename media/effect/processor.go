package effect

import (
	"fmt"
	"github.com/chwjbn/live-hub/glib"
	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"image"
	"path"
	"unsafe"
)

type EffectProcessor struct {
	mWidth         int
	mHeight        int
	mGLWindow      *glfw.Window
	mVertices      []float32
	mIndices       []uint32
	mVAO           uint32
	mShaderProgram *GlProgram
	mDstPixelData  []uint8
	mDstTexture    *GlTexture
}

func NewEffectProcessor() (*EffectProcessor, error) {

	pThis := new(EffectProcessor)
	pThis.mWidth = 10000
	pThis.mHeight = 10000

	xErr := pThis.init()

	if xErr != nil {
		return nil, xErr
	}

	return pThis, nil

}

func (e *EffectProcessor) ProcessAvFrameImage(frameImg *image.RGBA, imgRotate int) (*image.RGBA, error) {

	var xErr error

	e.mWidth = frameImg.Rect.Dx()
	e.mHeight = frameImg.Rect.Dy()

	e.mGLWindow.SetSize(e.mWidth, e.mHeight)

	gl.Viewport(0, 0, int32(e.mWidth), int32(e.mHeight))

	e.mDstTexture.SetImage(frameImg)

	//渲染
	gl.ClearColor(0, 0, 0, 1.0)   //状态设置
	gl.Clear(gl.COLOR_BUFFER_BIT) //状态使用

	e.mShaderProgram.Use()

	playTime := glfw.GetTime()
	gl.Uniform1f(e.mShaderProgram.GetUniformLocation("iPlayTime"), float32(playTime))

	e.mDstTexture.Bind(gl.TEXTURE0)
	e.mDstTexture.SetUniform(e.mShaderProgram.GetUniformLocation("iTexture0"))

	gl.BindVertexArray(e.mVAO)
	gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, unsafe.Pointer(nil))
	gl.BindVertexArray(0)

	e.mDstPixelData = make([]uint8, e.mWidth*e.mHeight*4)
	gl.ReadPixels(0, 0, int32(e.mWidth), int32(e.mHeight), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(e.mDstPixelData))

	srcImg := image.NewRGBA(image.Rect(0, 0, e.mWidth, e.mHeight))
	for y := 0; y < e.mHeight; y++ {
		for x := 0; x < e.mWidth; x++ {
			i := (y*e.mWidth + x) * 4
			srcImg.Pix[i] = e.mDstPixelData[i]
			srcImg.Pix[i+1] = e.mDstPixelData[i+1]
			srcImg.Pix[i+2] = e.mDstPixelData[i+2]
			srcImg.Pix[i+3] = e.mDstPixelData[i+3]
		}
	}

	e.mDstTexture.UnBind()
	e.mGLWindow.SwapBuffers()

	return srcImg, xErr

}

func (e *EffectProcessor) initData() error {
	var xErr error

	e.mVertices = []float32{
		// top left
		-1.0 * -1.0, -1.0 * 1.0, 0.0, // position
		1.0, 0.0, 0.0, // Color
		1.0, 0.0, // texture coordinates

		// top right
		-1.0 * 1.0, -1.0 * 1.0, 0.0,
		0.0, 1.0, 0.0,
		0.0, 0.0,

		// bottom right
		-1.0 * 1.0, -1.0 * -1.0, 0.0,
		0.0, 0.0, 1.0,
		0.0, 1.0,

		// bottom left
		-1.0 * -1.0, -1.0 * -1.0, 0.0,
		1.0, 1.0, 1.0,
		1.0, 1.0,
	}

	e.mIndices = []uint32{
		// rectangle
		0, 1, 2, // top triangle
		0, 2, 3, // bottom triangle
	}

	gl.GenVertexArrays(1, &e.mVAO)

	var VBO uint32
	gl.GenBuffers(1, &VBO)

	var EBO uint32
	gl.GenBuffers(1, &EBO)

	gl.BindVertexArray(e.mVAO)

	gl.BindBuffer(gl.ARRAY_BUFFER, VBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(e.mVertices)*4, gl.Ptr(e.mVertices), gl.STATIC_DRAW)

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, EBO)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(e.mIndices)*4, gl.Ptr(e.mIndices), gl.STATIC_DRAW)

	var stride int32 = 3*4 + 3*4 + 2*4
	var offset = 0

	// position
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, stride, uintptr(offset))
	gl.EnableVertexAttribArray(0)
	offset += 3 * 4

	// color
	gl.VertexAttribPointerWithOffset(1, 3, gl.FLOAT, false, stride, uintptr(offset))
	gl.EnableVertexAttribArray(1)
	offset += 3 * 4

	// texture position
	gl.VertexAttribPointerWithOffset(2, 2, gl.FLOAT, false, stride, uintptr(offset))
	gl.EnableVertexAttribArray(2)
	offset += 2 * 4

	// unbind the VAO (safe practice so we don't accidentally (mis)configure it later)
	gl.BindVertexArray(0)

	var glErr error
	e.mShaderProgram, glErr = e.getShaderProgram("soul")
	if glErr != nil {
		xErr = fmt.Errorf("getShaderProgram error:[%v]", glErr.Error())
		return xErr
	}

	e.mDstTexture, glErr = NewGlTexture()
	if glErr != nil {
		xErr = fmt.Errorf("NewGlTexture error:[%v]", glErr.Error())
		return xErr
	}

	return xErr
}

func (e *EffectProcessor) initGLFW() error {

	var xErr error

	glErr := glfw.Init()
	if xErr != nil {
		xErr = fmt.Errorf("glfw.Init error:[%v]", glErr.Error())
		return xErr
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.Decorated, glfw.False)
	glfw.WindowHint(glfw.Visible, glfw.False)

	e.mGLWindow, glErr = glfw.CreateWindow(e.mWidth, e.mHeight, "EffectProcessor", nil, nil)
	if xErr != nil {
		xErr = fmt.Errorf("glfw.CreateWindow error:[%v]", glErr.Error())
		return xErr
	}

	if e.mGLWindow == nil {
		xErr = fmt.Errorf("glfw.CreateWindow faild")
		return xErr
	}

	e.mGLWindow.MakeContextCurrent()

	//窗口变化
	e.mGLWindow.SetFramebufferSizeCallback(func(w *glfw.Window, width int, height int) {
		gl.Viewport(0, 0, int32(width), int32(height))
	})

	return xErr

}

func (e *EffectProcessor) initOpenGL() error {

	var xErr error

	glErr := gl.Init()
	if glErr != nil {
		xErr = fmt.Errorf("gl.Init error:[%v]", glErr.Error())
		return xErr
	}

	gl.Viewport(0, 0, int32(e.mWidth), int32(e.mHeight))
	gl.ClearColor(0, 0, 0, 1)

	return xErr
}

func (e *EffectProcessor) init() error {

	var xErr error

	xErr = e.initGLFW()
	if xErr != nil {
		return xErr
	}

	xErr = e.initOpenGL()
	if xErr != nil {
		return xErr
	}

	xErr = e.initData()
	if xErr != nil {
		return xErr
	}

	return xErr

}

func (e *EffectProcessor) getShaderProgram(effectName string) (*GlProgram, error) {

	var shaderProgram *GlProgram
	var xErr error

	vertexShaderCode := e.readShaderCode(effectName, "vert")
	if len(vertexShaderCode) < 1 {
		xErr = fmt.Errorf("missing vertex code in effect=[%v]", effectName)
		return shaderProgram, xErr
	}

	fragmentShaderCode := e.readShaderCode(effectName, "frag")
	if len(fragmentShaderCode) < 1 {
		xErr = fmt.Errorf("missing fragment code in effect=[%v]", effectName)
		return shaderProgram, xErr
	}

	vertexShader, shaderErr := NewGlShader(vertexShaderCode, gl.VERTEX_SHADER)
	if shaderErr != nil {
		xErr = fmt.Errorf("compile vertex shader error:%v", shaderErr.Error())
		return shaderProgram, xErr
	}

	fragmentShader, shaderErr := NewGlShader(fragmentShaderCode, gl.FRAGMENT_SHADER)
	if shaderErr != nil {
		xErr = fmt.Errorf("compile fragment shader error:%v", shaderErr.Error())
		return shaderProgram, xErr
	}

	shaderProgram, progErr := NewGlProgram(vertexShader, fragmentShader)
	if progErr != nil {
		xErr = fmt.Errorf("create program error:%v", progErr.Error())
		return shaderProgram, xErr
	}

	return shaderProgram, xErr

}

func (e *EffectProcessor) readShaderCode(effectName string, shaderType string) string {

	srcCode := ""

	codeFilePath := path.Join(glib.AppBaseDir(), "data", "effect", effectName, fmt.Sprintf("code.%s", shaderType))
	if !glib.FileExists(codeFilePath) {
		return srcCode
	}

	srcCode = glib.FileReadAllText(codeFilePath)

	return srcCode

}
