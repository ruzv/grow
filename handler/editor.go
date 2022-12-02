package handler

import (
	"image/color"

	"private/grow/blob"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"golang.org/x/image/font/basicfont"
)

type EditorMode string

const (
	EditorModeNone         EditorMode = "none"
	EditorModeAddNode      EditorMode = "add_node"
	EditorModeConnectNodes EditorMode = "connect_nodes"
	EditorModeAddUnit      EditorMode = "add_unit"
)

type Editor struct {
	win  *pixelgl.Window
	view *View
	blob *blob.Blob

	mode        EditorMode
	addNodeType blob.NodeType
	target      int
	targetSet   bool

	buttons    *editorButtons
	allbuttons []*button
}

type editorButtons struct {
	addNode                       *button
	addNodeNone                   *button
	addNodeMosFarm                *button
	addNodeMosFermentationChamber *button
	addNodeMushroomFarm           *button
	addNodeStorage                *button
	connectNodes                  *button
	addUnit                       *button
	removeUnits                   *button
	removeResources               *button
}

func NewEditor(win *pixelgl.Window, view *View, b *blob.Blob) *Editor {
	e := &Editor{
		win:         win,
		view:        view,
		blob:        b,
		mode:        EditorModeNone,
		addNodeType: blob.NodeTypeNone,
		buttons: &editorButtons{
			addNode: newButton(
				pixel.V(10, 6),
				"add node",
				false,
			),
			addNodeNone: newButton(
				pixel.V(7, 18),
				"none",
				true,
			),
			addNodeMosFarm: newButton(
				pixel.V(10, 30),
				"mos farm",
				true,
			),
			addNodeMosFermentationChamber: newButton(
				pixel.V(24, 42),
				"mos fermentation chamber",
				true,
			),
			addNodeMushroomFarm: newButton(
				pixel.V(15, 54),
				"mushroom farm",
				true,
			),
			addNodeStorage: newButton(
				pixel.V(10, 66),
				"storage",
				true,
			),
			connectNodes: newButton(
				pixel.V(58, 6),
				"connect nodes",
				false,
			),
			addUnit: newButton(
				pixel.V(123, 6),
				"add unit",
				false,
			),
			removeUnits: newButton(
				pixel.V(170, 6),
				"remove units",
				false,
			),
			removeResources: newButton(
				pixel.V(238, 6),
				"remove resources",
				false,
			),
		},
	}

	e.allbuttons = []*button{
		e.buttons.addNode,
		e.buttons.addNodeNone,
		e.buttons.addNodeMosFarm,
		e.buttons.addNodeMosFermentationChamber,
		e.buttons.addNodeMushroomFarm,
		e.buttons.addNodeStorage,
		e.buttons.connectNodes,
		e.buttons.addUnit,
		e.buttons.removeUnits,
		e.buttons.removeResources,
	}

	e.buttons.addNode.onClick = func(_ *button) {
		e.buttons.addNodeNone.hidden = !e.buttons.addNodeNone.hidden
		e.buttons.addNodeMosFarm.hidden = !e.buttons.addNodeMosFarm.hidden
		e.buttons.addNodeMosFermentationChamber.hidden = !e.buttons.addNodeMosFermentationChamber.hidden
		e.buttons.addNodeMushroomFarm.hidden = !e.buttons.addNodeMushroomFarm.hidden
		e.buttons.addNodeStorage.hidden = !e.buttons.addNodeStorage.hidden
	}

	e.buttons.addNodeNone.onClick = func(_ *button) {
		e.buttons.addNodeNone.hidden = true
		e.buttons.addNodeMosFarm.hidden = true
		e.buttons.addNodeMosFermentationChamber.hidden = true
		e.buttons.addNodeMushroomFarm.hidden = true
		e.buttons.addNodeStorage.hidden = true

		e.mode = EditorModeAddNode
		e.addNodeType = blob.NodeTypeNone
	}

	e.buttons.addNodeMosFarm.onClick = func(_ *button) {
		e.buttons.addNodeNone.hidden = true
		e.buttons.addNodeMosFarm.hidden = true
		e.buttons.addNodeMosFermentationChamber.hidden = true
		e.buttons.addNodeMushroomFarm.hidden = true
		e.buttons.addNodeStorage.hidden = true

		e.mode = EditorModeAddNode
		e.addNodeType = blob.NodeTypeMossFarm
	}

	e.buttons.addNodeMosFermentationChamber.onClick = func(_ *button) {
		e.buttons.addNodeNone.hidden = true
		e.buttons.addNodeMosFarm.hidden = true
		e.buttons.addNodeMosFermentationChamber.hidden = true
		e.buttons.addNodeMushroomFarm.hidden = true
		e.buttons.addNodeStorage.hidden = true

		e.mode = EditorModeAddNode
		e.addNodeType = blob.NodeTypeMossFermentationChamber
	}

	e.buttons.addNodeMushroomFarm.onClick = func(_ *button) {
		e.buttons.addNodeNone.hidden = true
		e.buttons.addNodeMosFarm.hidden = true
		e.buttons.addNodeMosFermentationChamber.hidden = true
		e.buttons.addNodeMushroomFarm.hidden = true
		e.buttons.addNodeStorage.hidden = true

		e.mode = EditorModeAddNode
		e.addNodeType = blob.NodeTypeMushroomFarm
	}

	e.buttons.addNodeStorage.onClick = func(_ *button) {
		e.buttons.addNodeNone.hidden = true
		e.buttons.addNodeMosFarm.hidden = true
		e.buttons.addNodeMosFermentationChamber.hidden = true
		e.buttons.addNodeMushroomFarm.hidden = true
		e.buttons.addNodeStorage.hidden = true

		e.mode = EditorModeAddNode
		e.addNodeType = blob.NodeTypeStorage
	}

	e.buttons.connectNodes.onClick = func(_ *button) {
		e.buttons.addNodeNone.hidden = true
		e.buttons.addNodeMosFarm.hidden = true
		e.buttons.addNodeMosFermentationChamber.hidden = true
		e.buttons.addNodeMushroomFarm.hidden = true
		e.buttons.addNodeStorage.hidden = true

		e.mode = EditorModeConnectNodes
	}

	e.buttons.addUnit.onClick = func(_ *button) {
		e.buttons.addNodeNone.hidden = true
		e.buttons.addNodeMosFarm.hidden = true
		e.buttons.addNodeMosFermentationChamber.hidden = true
		e.buttons.addNodeMushroomFarm.hidden = true
		e.buttons.addNodeStorage.hidden = true

		e.mode = EditorModeAddUnit
	}

	e.buttons.removeUnits.onClick = func(_ *button) {
		e.blob.RemoveUnits()
	}

	e.buttons.removeResources.onClick = func(_ *button) {
		e.blob.RemoveResources()
	}

	return e
}

func (e *Editor) Update() {
	for _, button := range e.allbuttons {
		if button.clicked {
			button.clickCooldown--
			if button.clickCooldown <= 0 {
				button.clicked = false
				button.color = color.RGBA{50, 50, 50, 255}
			}
		}

		if button.hidden {
			continue
		}

		if e.win.JustPressed(pixelgl.MouseButtonLeft) &&
			button.rect.Contains(e.win.MousePosition()) {
			button.click()
			return
		}
	}

	if e.win.JustPressed(pixelgl.KeyEscape) {
		e.mode = EditorModeNone
		e.buttons.addNodeNone.hidden = true
	}

	if e.win.JustPressed(pixelgl.MouseButtonLeft) {
		switch e.mode {
		case EditorModeAddNode:
			e.blob.AddNode(e.view.MousePos(), e.addNodeType)
		case EditorModeConnectNodes:
			if !e.targetSet {
				id, err := e.blob.GetClosestNode(e.view.MousePos())
				if err == nil {
					e.target = id
					e.targetSet = true
				}

				return
			}

			id, err := e.blob.GetClosestNode(e.view.MousePos())
			if err == nil {
				e.blob.Connect(e.target, id)
			}

			e.targetSet = false
		case EditorModeAddUnit:
			id, err := e.blob.GetClosestNode(e.view.MousePos())
			if err == nil {
				e.blob.AddUnit(id)
			}
		}
	}
}

func (e *Editor) Render() {
	// TODO: render indicators for each editor mode. ghost node for add node,
	// etc.

	e.view.UndoTransform()
	for _, button := range e.allbuttons {
		button.render(e.win)
	}
	e.view.Transform()
}

type button struct {
	pos           pixel.Vec
	color         color.Color
	text          *text.Text
	onClick       func(*button)
	hidden        bool
	rect          pixel.Rect
	clicked       bool
	clickCooldown int
}

func newButton(
	pos pixel.Vec,
	buttonText string,
	hidden bool,
) *button {
	text := text.New(
		pos.Add(pos),
		text.NewAtlas(basicfont.Face7x13, text.ASCII),
	)

	text.Write([]byte(buttonText))

	rect := text.Bounds()
	rect = rect.Resized(rect.Center(), rect.Size().Scaled(1.5))

	b := &button{
		pos:     pos,
		color:   color.RGBA{50, 50, 50, 255},
		text:    text,
		onClick: nil,
		hidden:  hidden,
		rect:    rect,
	}

	return b
}

func (b *button) render(win *pixelgl.Window) {
	if b.hidden {
		return
	}

	imd := imdraw.New(nil)
	imd.Color = b.color

	vertices := b.rect.Vertices()

	imd.Push(vertices[0])
	imd.Push(vertices[1])
	imd.Push(vertices[2])
	imd.Push(vertices[3])

	imd.Polygon(0)

	imd.Draw(win)
	b.text.Draw(win, pixel.IM)
}

func (b *button) click() {
	b.color = color.RGBA{40, 40, 40, 255}
	b.clickCooldown = 6
	b.clicked = true
	b.onClick(b)
}
