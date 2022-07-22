package blob

import (
	"errors"
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"private/grow/config"
	"private/grow/render"

	"github.com/faiface/pixel"
)

type ResourceType string

const (
	ResourceTypeMoss ResourceType = "moss"
)

type Resource struct {
	id           int
	resourceType ResourceType
	pos          pixel.Vec
}

type ResourceJSON struct {
	ID           int          `json:"id"`
	ResourceType ResourceType `json:"resource_type"`
	Pos          pixel.Vec    `json:"pos"`
}

func NewResource(rj *ResourceJSON) *Resource {
	r := &Resource{}

	r.id = rj.ID
	r.resourceType = rj.ResourceType
	r.pos = rj.Pos

	return r
}

func (r *Resource) ToJSON() *ResourceJSON {
	return &ResourceJSON{
		ID:           r.id,
		ResourceType: r.resourceType,
		Pos:          r.pos,
	}
}

func (r *Resource) Render(rend *render.Renderer) {
	switch r.resourceType {
	case ResourceTypeMoss:
		rend.Circle(r.pos, color.RGBA{153, 255, 102, 255}, 3, 0)
	}
}

type NodeType string

const (
	NodeTypeNone                    NodeType = "none"
	NodeTypeMossFarm                NodeType = "moss_farm"
	NodeTypeMossFermentationChamber NodeType = "moss_fermentation_chamber"
)

type Node struct {
	id        int
	pos       pixel.Vec
	nodeType  NodeType
	resources map[int]*Resource
	consumes  []ResourceType
	conf      *config.NodeConfig
	blob      *Blob
}

type NodeJSON struct {
	ID        int                   `json:"id"`
	Pos       pixel.Vec             `json:"pos"`
	Type      NodeType              `json:"type"`
	Resources map[int]*ResourceJSON `json:"resources"`
}

func GetNodeConfig(nodeType NodeType, conf *config.Config) *config.NodeConfig {
	switch nodeType {
	case NodeTypeNone:
		return &conf.Nodes.None
	case NodeTypeMossFarm:
		return &conf.Nodes.MossFarm
	case NodeTypeMossFermentationChamber:
		return &conf.Nodes.MossFermentationChamber
	}

	return nil
}

func NewNode(nj *NodeJSON, b *Blob, conf *config.Config) *Node {
	n := &Node{}

	n.id = nj.ID
	n.pos = nj.Pos
	n.nodeType = nj.Type

	n.resources = make(map[int]*Resource)
	for id, res := range nj.Resources {
		n.resources[id] = NewResource(res)
	}

	n.conf = GetNodeConfig(n.nodeType, conf)
	n.blob = b

	return n
}

func (n *Node) ToJSON() *NodeJSON {
	nj := &NodeJSON{
		ID:   n.id,
		Pos:  n.pos,
		Type: n.nodeType,
	}

	nj.Resources = make(map[int]*ResourceJSON)
	for id, res := range n.resources {
		nj.Resources[id] = res.ToJSON()
	}

	return nj
}

func (n *Node) Render(rend *render.Renderer) {
	switch n.nodeType {
	case NodeTypeNone:
		rend.Circle(n.pos, color.RGBA{255, 255, 255, 255}, n.conf.Radius, 0)
	case NodeTypeMossFarm:
		rend.Circle(n.pos, color.RGBA{0, 102, 0, 255}, n.conf.Radius, 0)
		rend.Circle(n.pos, color.RGBA{0, 153, 0, 255}, n.conf.Radius, 3)
	case NodeTypeMossFermentationChamber:
		rend.Circle(n.pos, color.RGBA{0, 153, 153, 255}, n.conf.Radius, 0)
		rend.Circle(n.pos, color.RGBA{0, 102, 102, 255}, n.conf.Radius, 3)
	}

	for _, r := range n.resources {
		r.Render(rend)
	}

	rend.Text(n.pos, color.RGBA{255, 0, 0, 255}, fmt.Sprintf("%d", n.id), 1)
}

func (n *Node) RandPosInNode() pixel.Vec {
	return n.pos.Add(
		pixel.V(n.conf.Radius, 0).
			Scaled(rand.Float64() * 0.9).
			Rotated(rand.Float64() * 2 * math.Pi),
	)
}

func (n *Node) AddResource(resourceType ResourceType) (int, error) {
	if len(n.resources) >= n.conf.ResourceCapacity {
		return 0, errors.New("resource capacity reached")
	}

	id := n.blob.resourcesIdentifier
	n.blob.resourcesIdentifier++

	r := &Resource{
		id:           id,
		resourceType: resourceType,
		pos:          n.RandPosInNode(),
	}

	n.resources[id] = r

	return id, nil
}

func (n *Node) CanConsume(resourceType ResourceType) bool {
	for _, c := range n.conf.Consumes {
		if ResourceType(c) == resourceType {
			return true
		}
	}

	return false
}
