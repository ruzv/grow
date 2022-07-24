package blob

import (
	"errors"
	"math"
	"math/rand"

	"private/grow/render"

	"github.com/faiface/pixel"
)

type ResourceType string

const (
	ResourceTypeNone ResourceType = ""
	ResourceTypeMoss ResourceType = "moss"
)

type ResourceConfig struct {
	Graphics []*render.Primitive
}

func (resourceType ResourceType) Render(
	rend *render.Renderer,
	conf *BlobConfig,
	positions ...pixel.Vec,
) {
	for _, pos := range positions {
		rend.Primitives(pos, conf.Resources[resourceType].Graphics...)
	}
}

type NodeType string

const (
	NodeTypeNone                    NodeType = "none"
	NodeTypeMossFarm                NodeType = "moss_farm"
	NodeTypeMossFermentationChamber NodeType = "moss_fermentation_chamber"
)

type NodeConfig struct {
	Radius           float64              `json:"radius"`
	ResourceCapacity map[ResourceType]int `json:"resource_capacity"`
	Consumes         []ResourceType       `json:"consumes"`
	Produces         []ResourceType       `json:"produces"`
	Jobs             []JobType            `json:"jobs"`
	Graphics         []*render.Primitive  `json:"graphics"`
}

type Node struct {
	id                 int
	pos                pixel.Vec
	nodeType           NodeType
	resources          map[ResourceType][]pixel.Vec
	productionProgress float64
	conf               *NodeConfig
	blob               *Blob
}

type NodeJSON struct {
	ID                 int                          `json:"id"`
	Pos                pixel.Vec                    `json:"pos"`
	NodeType           NodeType                     `json:"node_type"`
	Resources          map[ResourceType][]pixel.Vec `json:"resources"`
	ProductionProgress float64                      `json:"production_progress"`
}

func NewNode(nj *NodeJSON, b *Blob, conf *BlobConfig) *Node {
	n := &Node{
		id:                 nj.ID,
		pos:                nj.Pos,
		nodeType:           nj.NodeType,
		resources:          nj.Resources,
		productionProgress: nj.ProductionProgress,
	}

	if n.resources == nil {
		n.resources = make(map[ResourceType][]pixel.Vec)
	}

	n.conf = conf.Nodes[n.nodeType]
	n.blob = b

	return n
}

func (n *Node) ToJSON() *NodeJSON {
	nj := &NodeJSON{
		ID:                 n.id,
		Pos:                n.pos,
		NodeType:           n.nodeType,
		Resources:          n.resources,
		ProductionProgress: n.productionProgress,
	}

	return nj
}

func (n *Node) Jobs() []*Job {
	jobs := make([]*Job, 0, len(n.conf.Jobs))

	for _, jobType := range n.conf.Jobs {
		jobs = append(jobs, NewJob(
			&JobJSON{
				ID:      n.blob.jobsIdentifier,
				NodeID:  n.id,
				JobType: jobType,
			},
			n.blob.conf,
			n.blob,
		))

		n.blob.jobsIdentifier++
	}

	return jobs
}

func (n *Node) Render(rend *render.Renderer) {
	rend.Primitives(n.pos, n.conf.Graphics...)

	for resourceType, positions := range n.resources {
		resourceType.Render(rend, n.blob.conf, positions...)
	}

	// rend.Text(n.pos, color.RGBA{255, 0, 0, 255}, fmt.Sprintf("%d", n.id), 1)
}

func (n *Node) Update() {
	switch n.nodeType {
	case NodeTypeMossFermentationChamber:
		n.productionProgress += 0.1 // TODO: config
		if n.productionProgress > 40 {
			n.productionProgress = 0
			n.TakeResource(ResourceTypeMoss)
		}
	}
}

func (n *Node) RandPosInNode() pixel.Vec {
	return n.pos.Add(
		pixel.V(n.conf.Radius, 0).
			Scaled(rand.Float64() * 0.9).
			Rotated(rand.Float64() * 2 * math.Pi),
	)
}

func (n *Node) AddResource(resourceType ResourceType) error {
	if len(n.resources[resourceType]) >= n.conf.ResourceCapacity[resourceType] {
		return errors.New("resource capacity reached")
	}

	n.resources[resourceType] = append(
		n.resources[resourceType],
		n.RandPosInNode(),
	)

	return nil
}

func (n *Node) TakeResource(resourceType ResourceType) error {
	if len(n.resources[resourceType]) == 0 {
		return errors.New("no resource")
	}

	n.resources[resourceType] = n.resources[resourceType][1:]

	return nil
}

func (n *Node) Consumes() []ResourceType {
	consumes := make([]ResourceType, 0, len(n.conf.Consumes))

	for _, c := range n.conf.Consumes {
		consumes = append(consumes, ResourceType(c))
	}

	return consumes
}

func (n *Node) AvailableCapacity(resourceType ResourceType) int {
	return n.conf.ResourceCapacity[resourceType] -
		len(n.resources[resourceType])
}

func (n *Node) Resources(resourceType ResourceType) int {
	return len(n.resources[resourceType])
}

// func (n *Node) CanConsume() []ResourceType {
// 	var consumes []ResourceType

// 	for resourceType, resources := range n.resources {
// 		if len(resources) < n.conf.ResourceCapacity[resourceType] {
// 			consumes = append(consumes, resourceType)
// 		}
// 	}

// 	return consumes
// }

// func (n *Node) HasResources() []ResourceType {
// 	var has []ResourceType

// 	for resourceType, resources := range n.resources {
// 		if len(resources) > 0 {
// 			has = append(has, resourceType)
// 		}
// 	}

// 	return has
// }

type JobType string

const JobTypeGrowMoss JobType = "grow_moss"

type JobConfig struct {
	ProducedResource ResourceType `json:"produced_resource"`
	ProductionSpeed  float64      `json:"production_speed"`

	// latter this will allow to add support for multiple produced resources,
	// required resources, ...
}

type Job struct {
	id      int
	nodeID  int
	jobType JobType

	conf *JobConfig
	blob *Blob
}

type JobJSON struct {
	ID      int     `json:"id"`
	NodeID  int     `json:"node_id"`
	JobType JobType `json:"job_type"`
}

func NewJob(jj *JobJSON, conf *BlobConfig, blob *Blob) *Job {
	if jj == nil {
		return nil
	}

	j := &Job{
		id:      jj.ID,
		nodeID:  jj.NodeID,
		jobType: jj.JobType,
		blob:    blob,
	}

	j.conf = conf.Jobs[j.jobType]

	return j
}

func (j *Job) ToJSON() *JobJSON {
	if j == nil {
		return nil
	}

	jj := &JobJSON{
		ID:      j.id,
		NodeID:  j.nodeID,
		JobType: j.jobType,
	}

	return jj
}

func (j *Job) Complete() error {
	return j.blob.Nodes[j.nodeID].AddResource(j.conf.ProducedResource)
}
