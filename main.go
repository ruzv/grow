package main

import (
	"image/color"
	"time"

	"private/grow/blob"
	"private/grow/config"
	"private/grow/handler"
	"private/grow/render"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/pkg/errors"
)

type Handler struct {
	win           *pixelgl.Window
	rend          *render.Renderer
	prevFrameTime time.Time
	frameDuration time.Duration
}

func NewHandler(fps int) (*Handler, error) {
	win, err := pixelgl.NewWindow(
		pixelgl.WindowConfig{
			Title:  "grow",
			Bounds: pixel.R(0, 0, 800, 800),
			VSync:  true,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create window")
	}

	win.SetSmooth(true)

	return &Handler{
		win:           win,
		rend:          render.NewRenderer(win),
		prevFrameTime: time.Now(),
		frameDuration: time.Second / time.Duration(fps),
	}, nil
}

func (h *Handler) FrameDelay() {
	now := time.Now()

	dt := h.frameDuration - now.Sub(h.prevFrameTime)

	time.Sleep(dt)

	h.prevFrameTime = now
}

func run() {
	h, err := NewHandler(60)
	if err != nil {
		panic(err)
	}

	conf, err := config.LoadConfig("config.json")
	if err != nil {
		panic(err)
	}

	save, err := config.LoadSave("save.json")
	if err != nil {
		panic(err)
	}

	b := blob.NewBlob(save.Blob, &conf.Blob, h.win)
	v := handler.NewView(save.View, &conf.View, h.win)
	e := handler.NewEditor(h.win, v, b)

	for !h.win.Closed() {
		h.win.Clear(color.RGBA{0, 0, 0, 255})
		b.Render()
		e.Render()

		b.Update()
		e.Update()
		v.Update()
		h.win.Update()
		h.FrameDelay()
	}

	err = config.RecordSave(
		"save.json",
		&config.Save{
			Blob: b.ToJSON(),
			View: v.ToJSON(),
		},
	)
	if err != nil {
		panic(err)
	}
}

func main() {
	pixelgl.Run(run)
}
