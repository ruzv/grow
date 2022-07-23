package main

import (
	"image/color"
	"time"

	"private/grow/blob"
	"private/grow/config"
	"private/grow/render"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"github.com/pkg/errors"
	"golang.org/x/image/font/basicfont"
)

type handler struct {
	win           *pixelgl.Window
	rend          *render.Renderer
	prevFrameTime time.Time
	frameDuration time.Duration
}

func NewHandler(fps int) (*handler, error) {
	win, err := pixelgl.NewWindow(
		pixelgl.WindowConfig{
			Title:  "grow",
			Bounds: pixel.R(0, 0, 600, 600),
			// VSync:  true,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create window")
	}

	win.SetSmooth(true)

	return &handler{
		win:           win,
		rend:          render.NewRenderer(win),
		prevFrameTime: time.Now(),
		frameDuration: time.Second / time.Duration(fps),
	}, nil
}

func (h *handler) FrameDelay() {
	now := time.Now()

	dt := h.frameDuration - now.Sub(h.prevFrameTime)

	time.Sleep(dt)

	h.prevFrameTime = now
}

type Button struct {
	pos           pixel.Vec
	width, height float64
	color         color.Color
	clickColor    color.Color
	renderColor   color.Color
	text          *text.Text
	onClick       func()
	rect          pixel.Rect
	hidden        bool
}

func (e *Editor) NewButton(
	pos pixel.Vec,
	width, height float64,
	color color.Color,
	clickColor color.Color,
	buttonText string,
	textColor color.Color,
	textPos pixel.Vec,
	hidden bool,
	onClick func(),
) *Button {
	button := &Button{
		pos:         pos,
		width:       width,
		height:      height,
		color:       color,
		clickColor:  clickColor,
		renderColor: color,
		text: text.New(
			pos.Add(textPos),
			text.NewAtlas(basicfont.Face7x13, text.ASCII),
		),
		hidden:  hidden,
		onClick: onClick,
		rect:    pixel.R(pos.X, pos.Y, pos.X+width, pos.Y+height),
	}

	button.text.Color = textColor
	button.text.Write([]byte(buttonText))

	e.buttons = append(e.buttons, button)

	return button
}

type EditorMode int

const (
	EditorModeNone EditorMode = iota
	EditorModeAddNode
	EditorModeConnectNodes
	EditorModeAddUnit
)

var EditorModeDescriptions = map[EditorMode]string{
	EditorModeNone:         "none",
	EditorModeAddNode:      "add node",
	EditorModeConnectNodes: "connect nodes",
	EditorModeAddUnit:      "add unit",
}

type Editor struct {
	h         *handler
	mode      EditorMode
	nodeType  blob.NodeType
	target    int
	targetSet bool

	buttons []*Button
}

func NewEditor(h *handler) *Editor {
	e := &Editor{
		h: h,
		// textBox: h.rend.NewTextBox(pixel.V(10, 10), 2),
		mode:     EditorModeNone,
		nodeType: blob.NodeTypeNone,
	}

	addNoneNode := e.NewButton(
		pixel.V(10, 40),
		50,
		26,
		color.RGBA{50, 50, 50, 255},
		color.RGBA{40, 40, 40, 255},
		"none",
		color.RGBA{255, 255, 255, 255},
		pixel.V(10, 10),
		true,
		nil,
	)

	addMossFarmNode := e.NewButton(
		pixel.V(65, 40),
		80,
		26,
		color.RGBA{50, 50, 50, 255},
		color.RGBA{40, 40, 40, 255},
		"moss farm",
		color.RGBA{255, 255, 255, 255},
		pixel.V(10, 10),
		true,
		nil,
	)

	addMossFerChamb := e.NewButton(
		pixel.V(150, 40),
		120,
		26,
		color.RGBA{50, 50, 50, 255},
		color.RGBA{40, 40, 40, 255},
		"moss fern chamb",
		color.RGBA{255, 255, 255, 255},
		pixel.V(10, 10),
		true,
		nil,
	)

	addNoneNode.onClick = func() {
		addNoneNode.hidden = true
		addMossFarmNode.hidden = true
		addMossFerChamb.hidden = true

		e.mode = EditorModeAddNode
		e.nodeType = blob.NodeTypeNone
	}

	addMossFarmNode.onClick = func() {
		addNoneNode.hidden = true
		addMossFarmNode.hidden = true
		addMossFerChamb.hidden = true

		e.mode = EditorModeAddNode
		e.nodeType = blob.NodeTypeMossFarm
	}

	addMossFerChamb.onClick = func() {
		addNoneNode.hidden = true
		addMossFarmNode.hidden = true
		addMossFerChamb.hidden = true

		e.mode = EditorModeAddNode
		e.nodeType = blob.NodeTypeMossFermentationChamber
	}

	e.NewButton(
		pixel.V(10, 10),
		70,
		26,
		color.RGBA{50, 50, 50, 255},
		color.RGBA{40, 40, 40, 255},
		"add node",
		color.RGBA{255, 255, 255, 255},
		pixel.V(10, 10),
		false,
		func() {
			addNoneNode.hidden = false
			addMossFarmNode.hidden = false
			addMossFerChamb.hidden = false
		},
	)

	e.NewButton(
		pixel.V(85, 10),
		70,
		26,
		color.RGBA{50, 50, 50, 255},
		color.RGBA{40, 40, 40, 255},
		"connect",
		color.RGBA{255, 255, 255, 255},
		pixel.V(10, 10),
		false,
		func() { e.mode = EditorModeConnectNodes },
	)

	e.NewButton(
		pixel.V(160, 10),
		70,
		26,
		color.RGBA{50, 50, 50, 255},
		color.RGBA{40, 40, 40, 255},
		"add unit",
		color.RGBA{255, 255, 255, 255},
		pixel.V(10, 10),
		false,
		func() { e.mode = EditorModeAddUnit },
	)

	// e.NewButton(pixel.V(200, 200), 100, 50, color.RGBA{255, 0, 0, 255},
	// "none")

	// e.textBox.Write(EditorModeDescriptions[e.mode])

	return e
}

func (e *Editor) Update(b *blob.Blob) {
	// switch {
	// case e.h.win.JustPressed(pixelgl.Key1):
	// 	e.mode = EditorModeNone
	// case e.h.win.JustPressed(pixelgl.Key2):
	// 	e.mode = EditorModeAddNode
	// case e.h.win.JustPressed(pixelgl.Key3):
	// 	e.mode = EditorModeConnectNodes
	// case e.h.win.JustPressed(pixelgl.Key4):
	// 	e.mode = EditorModeAddUnit
	if e.h.win.JustPressed(pixelgl.KeyEscape) {
		e.mode = EditorModeNone
	}

	if e.h.win.JustPressed(pixelgl.MouseButtonLeft) {
		switch e.mode {
		case EditorModeAddNode:
			b.AddNode(e.h.win.MousePosition(), e.nodeType)
		case EditorModeConnectNodes:
			if !e.targetSet {
				id, err := b.GetClosestNode(e.h.win.MousePosition())
				if err == nil {
					e.target = id
					e.targetSet = true
				}

				return
			}

			id, err := b.GetClosestNode(e.h.win.MousePosition())
			if err == nil {
				b.Connect(e.target, id)
			}

			e.targetSet = false
		case EditorModeAddUnit:
			id, err := b.GetClosestNode(e.h.win.MousePosition())
			if err == nil {
				b.AddUnit(id)
			}
		}
	}

	for _, button := range e.buttons {
		if button.hidden {
			continue
		}

		if e.h.win.JustPressed(pixelgl.MouseButtonLeft) &&
			button.rect.Contains(e.h.win.MousePosition()) {
			button.renderColor = button.clickColor
			button.onClick()
		} else {
			button.renderColor = button.color
		}
	}
}

func (e *Editor) Render() {
	for _, button := range e.buttons {
		if button.hidden {
			continue
		}

		imd := imdraw.New(nil)
		imd.Color = button.renderColor

		imd.Push(button.pos)
		imd.Push(button.pos.Add(pixel.V(button.width, 0)))
		imd.Push(button.pos.Add(pixel.V(button.width, button.height)))
		imd.Push(button.pos.Add(pixel.V(0, button.height)))

		imd.Polygon(0)

		imd.Draw(e.h.win)

		button.text.Draw(e.h.win, pixel.IM)
	}
}

func run() {
	h, err := NewHandler(30)
	if err != nil {
		panic(err)
	}

	e := NewEditor(h)

	conf, err := config.LoadConfig("config.json")

	b, err := blob.LoadBlob("blob.json", &conf.Blob)
	if err != nil {
		panic(err)
	}

	for !h.win.Closed() {

		h.win.Clear(color.RGBA{
			R: 0,
			G: 0,
			B: 0,
			A: 255,
		})

		b.Render(h.rend)

		e.Render()

		h.rend.Render()

		b.Update()

		e.Update(b)

		h.win.Update()

		h.FrameDelay()

	}

	b.Save("blob.json")
}

func main() {
	pixelgl.Run(run)
}
