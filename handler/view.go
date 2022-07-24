package handler

import (
	"math"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
)

type ViewConfig struct {
	PanSpeed  float64 `json:"pan_speed"`
	ZoomSpeed float64 `json:"zoom_speed"`
	MaxZoom   float64 `json:"max_zoom"`
	MinZoom   float64 `json:"min_zoom"`
}

type View struct {
	win            *pixelgl.Window
	pos            pixel.Vec
	zoom           float64
	transformation pixel.Matrix

	conf *ViewConfig
}

type ViewJSON struct {
	Pos            pixel.Vec    `json:"pos"`
	Zoom           float64      `json:"zoom"`
	Transformation pixel.Matrix `json:"transformation"`
}

func NewView(vj *ViewJSON, conf *ViewConfig, win *pixelgl.Window) *View {
	v := &View{
		win:            win,
		pos:            vj.Pos,
		zoom:           vj.Zoom,
		transformation: vj.Transformation,
		conf:           conf,
	}

	v.update()

	return v
}

func (v *View) ToJSON() *ViewJSON {
	return &ViewJSON{
		Pos:            v.pos,
		Zoom:           v.zoom,
		Transformation: v.transformation,
	}
}

func (v *View) Update() {
	updated := v.getUpdates()
	if !updated {
		return
	}

	v.update()
}

func (v *View) getUpdates() bool {
	speed := v.conf.PanSpeed / v.zoom

	var updated bool

	if v.win.Pressed(pixelgl.KeyW) { // up
		v.pos.Y += speed
		updated = true
	}
	if v.win.Pressed(pixelgl.KeyS) { // down
		v.pos.Y -= speed
		updated = true

	}
	if v.win.Pressed(pixelgl.KeyA) { // left
		v.pos.X -= speed
		updated = true

	}
	if v.win.Pressed(pixelgl.KeyD) { // right
		v.pos.X += speed
		updated = true
	}

	scroll := v.win.MouseScroll().Y
	if scroll > 0 {
		if v.zoom < v.conf.MaxZoom {
			v.zoom *= math.Pow(v.conf.ZoomSpeed, scroll)
			updated = true
		}
	} else if scroll < 0 {
		if v.zoom > v.conf.MinZoom {
			v.zoom *= math.Pow(v.conf.ZoomSpeed, scroll)
			updated = true
		}
	}

	return updated
}

func (v *View) update() {
	v.transformation = pixel.IM.
		Scaled(v.pos, v.zoom).
		Moved(v.win.Bounds().Center().Sub(v.pos))
	v.Transform()
}

func (v *View) Transform() {
	v.win.SetMatrix(v.transformation)
}

func (v *View) UndoTransform() {
	v.win.SetMatrix(pixel.IM)
}

func (v *View) MousePos() pixel.Vec {
	return v.transformation.Unproject(v.win.MousePosition())
}
