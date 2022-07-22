package render

import (
	"image/color"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"golang.org/x/image/font/basicfont"
)

// Renderer is a convinient wrapper around imdraw.IMDraw to allow for easy
// drawing of primitives such as lines and circles.
type Renderer struct {
	win       *pixelgl.Window
	atlas     *text.Atlas
	textBoxes []*TextBox
}

func NewRenderer(win *pixelgl.Window) *Renderer {
	return &Renderer{
		win:   win,
		atlas: text.NewAtlas(basicfont.Face7x13, text.ASCII),
	}
}

func (r *Renderer) Render() {
	for _, tb := range r.textBoxes {
		tb.writer.Draw(r.win, pixel.IM.Scaled(tb.writer.Orig, tb.scale))
	}
}

func (r *Renderer) Circle(
	pos pixel.Vec,
	color color.Color,
	radius float64,
	thickness float64,
) {
	imd := imdraw.New(nil)

	imd.Color = color
	imd.Push(pos)
	imd.Circle(radius, thickness)

	imd.Draw(r.win)
}

func (r *Renderer) Line(
	startPos, endPos pixel.Vec,
	color color.Color,
	thickness float64,
) {
	imd := imdraw.New(nil)

	imd.Color = color
	imd.Push(startPos, endPos)
	imd.Line(thickness)

	imd.Draw(r.win)
}

func (r *Renderer) Text(
	pos pixel.Vec,
	color color.Color,
	data string,
	scale float64,
) {
	writer := text.New(pos, r.atlas)
	writer.Color = color
	writer.Write([]byte(data))
	writer.Draw(r.win, pixel.IM.Scaled(writer.Orig, scale))
}

type TextBox struct {
	writer *text.Text
	scale  float64
}

func (r *Renderer) NewTextBox(pos pixel.Vec, scale float64) *TextBox {
	tb := &TextBox{
		writer: text.New(pos, r.atlas),
		scale:  scale,
	}

	r.textBoxes = append(r.textBoxes, tb)

	return tb
}

func (tb *TextBox) Write(data string) {
	tb.writer.Write([]byte(data))
}

func (tb *TextBox) Update(data string) {
	tb.writer.Clear()
	tb.Write(data)
}
