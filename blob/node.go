package blob

import (
	"fmt"
	"image/color"

	"private/grow/render"

	"github.com/faiface/pixel"
)

type NodeType int

const (
	NodeTypeNone NodeType = iota
	NodeTypeMossFarm
)

type Node struct {
	id       int
	pos      pixel.Vec
	nodeType NodeType
}

type NodeJSON struct {
	ID   int       `json:"id"`
	Pos  pixel.Vec `json:"pos"`
	Type NodeType  `json:"type"`
}

func NewNode(nj *NodeJSON) *Node {
	n := &Node{}

	n.id = nj.ID
	n.pos = nj.Pos
	n.nodeType = nj.Type

	return n
}

func (n *Node) ToJSON() *NodeJSON {
	return &NodeJSON{
		ID:   n.id,
		Pos:  n.pos,
		Type: n.nodeType,
	}
}

func (n *Node) Render(rend *render.Renderer) {
	switch n.nodeType {
	case NodeTypeNone:
		rend.Circle(n.pos, color.RGBA{255, 255, 255, 255}, 20, 0)
	case NodeTypeMossFarm:
		rend.Circle(n.pos, color.RGBA{0, 102, 0, 255}, 30, 0)
		rend.Circle(n.pos, color.RGBA{0, 153, 0, 255}, 28, 3)

	}

	rend.Text(n.pos, color.RGBA{255, 0, 0, 255}, fmt.Sprintf("%d", n.id), 1)
}

func (n *Node) RandPosInNode() pixel.Vec {
	// n.Pos.Rotated()
	return pixel.Vec{}
}
