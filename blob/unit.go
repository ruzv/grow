package blob

import (
	"fmt"
	"image/color"

	"private/grow/render"

	"github.com/faiface/pixel"
)

type ProcedureStepType string

const (
	DoNothing      ProcedureStepType = "do_nothing"
	Wander         ProcedureStepType = "wander"
	TraverseTo     ProcedureStepType = "traverse_to"
	Traverse       ProcedureStepType = "traverse"
	StartCarry     ProcedureStepType = "start_carry"
	StartLerp      ProcedureStepType = "start_lerp"
	PickUpResource ProcedureStepType = "pick_up_resource"
	DropResource   ProcedureStepType = "drop_resource"
	FindConsumer   ProcedureStepType = "find_consumer"
	FindJob        ProcedureStepType = "find_job"
	DoJob          ProcedureStepType = "do_job"
	FindTask       ProcedureStepType = "find_task"
	FindFood       ProcedureStepType = "find_food"
	Eat            ProcedureStepType = "eat"
)

type ProcedureStep struct {
	stepType     ProcedureStepType
	nodeID       int
	resourceType ResourceType
}

type ProcedureStepJSON struct {
	StepType     ProcedureStepType `json:"step_type"`
	NodeID       int               `json:"node_id"`
	ResourceType ResourceType      `json:"resource_type"`
}

func NewProcedureStep(psj *ProcedureStepJSON) *ProcedureStep {
	return &ProcedureStep{
		stepType:     psj.StepType,
		nodeID:       psj.NodeID,
		resourceType: psj.ResourceType,
	}
}

func (ps *ProcedureStep) ToJSON() *ProcedureStepJSON {
	return &ProcedureStepJSON{
		StepType:     ps.stepType,
		NodeID:       ps.nodeID,
		ResourceType: ps.resourceType,
	}
}

type UnitConfig struct {
	TraversalSpeed float64 `json:"traversal_speed"`
	HungerRate     float64 `json:"hunger_rate"`
	MaxHunger      float64 `json:"max_hunger"`
}

type Unit struct {
	id int

	procedure []*ProcedureStep
	nodeID    int

	traversingPath       []int
	traversingConnection *Connection
	traversingStep       int
	traversingProgress   float64

	stationaryPos          pixel.Vec
	stationaryTarget       pixel.Vec
	stationaryLerpProgress float64

	resource ResourceType

	job         *Job
	jobProgress float64

	hunger float64

	blob *Blob
	conf *UnitConfig
}

type UnitJSON struct {
	ID        int                  `json:"id"`
	Procedure []*ProcedureStepJSON `json:"procedure"`
	NodeID    int                  `json:"node"`

	TraversingPath       []int       `json:"traversing_path"`
	TraversingConnection *Connection `json:"traversing_connection"`
	TraversingStep       int         `json:"traversing_step"`
	TraversingProgress   float64     `json:"traversing_progress"`

	StationaryPos          pixel.Vec `json:"stationary_pos"`
	StationaryTarget       pixel.Vec `json:"stationary_target"`
	StationaryLerpProgress float64   `json:"stationary_lerp_progress"`

	Resource ResourceType `json:"resource"`

	Job         *JobJSON `json:"job"`
	JobProgress float64  `json:"job_progress"`

	Hunger float64 `json:"hunger"`
}

func NewUnit(uj *UnitJSON, blob *Blob) *Unit {
	u := &Unit{
		id:        uj.ID,
		procedure: make([]*ProcedureStep, 0, len(uj.Procedure)),
		nodeID:    uj.NodeID,

		traversingPath:       uj.TraversingPath,
		traversingConnection: uj.TraversingConnection,
		traversingStep:       uj.TraversingStep,
		traversingProgress:   uj.TraversingProgress,

		stationaryPos:          uj.StationaryPos,
		stationaryTarget:       uj.StationaryTarget,
		stationaryLerpProgress: uj.StationaryLerpProgress,

		resource: uj.Resource,

		job:         NewJob(uj.Job, blob.conf, blob),
		jobProgress: uj.JobProgress,

		hunger: uj.Hunger,

		blob: blob,
		conf: blob.conf.Unit,
	}

	for _, p := range uj.Procedure {
		u.procedure = append(u.procedure, NewProcedureStep(p))
	}

	return u
}

func (u *Unit) ToJSON() *UnitJSON {
	uj := &UnitJSON{
		ID:        u.id,
		Procedure: make([]*ProcedureStepJSON, 0, len(u.procedure)),
		NodeID:    u.nodeID,

		TraversingPath:       u.traversingPath,
		TraversingConnection: u.traversingConnection,
		TraversingStep:       u.traversingStep,
		TraversingProgress:   u.traversingProgress,

		StationaryPos:          u.stationaryPos,
		StationaryTarget:       u.stationaryTarget,
		StationaryLerpProgress: u.stationaryLerpProgress,

		Resource: u.resource,

		Job:         u.job.ToJSON(),
		JobProgress: u.jobProgress,

		Hunger: u.hunger,
	}

	for _, p := range u.procedure {
		uj.Procedure = append(uj.Procedure, p.ToJSON())
	}

	return uj
}

func (u *Unit) Render(rend *render.Renderer) {
	pos := u.Pos()

	rend.Circle(pos, color.RGBA{50, 50, 50, 255}, 6, 0)

	if u.resource != ResourceTypeNone {
		u.resource.Render(rend, u.blob.conf, pos)
	}
}

func (u *Unit) Pos() pixel.Vec {
	switch u.CurrentProcedureStep().stepType {

	case DoJob:
		return u.stationaryPos

	case Traverse:
		start := u.blob.Nodes[u.nodeID].pos
		target := u.blob.Nodes[u.traversingConnection.Nodes.Opposite(u.nodeID)].pos

		return start.Add(target.Sub(start).Scaled(
			u.traversingProgress / u.traversingConnection.Length,
		))
	}

	return u.blob.Nodes[u.nodeID].pos
}

func (u *Unit) Update() {
	u.hunger += u.conf.HungerRate

	switch u.CurrentProcedureStep().stepType {
	case DoNothing:
		// do nothing
	case FindTask:
		if u.hunger > u.conf.MaxHunger {
			u.Die()
			return
		}

		if u.hunger > 100 {
			u.SetCurrentProcedureStep(FindFood)
			return
		}

		u.SetCurrentProcedureStep(FindJob)
	case Wander:
		var nodes []*Node
		for _, node := range u.blob.Nodes {
			if node.id == u.nodeID {
				continue
			}
			nodes = append(nodes, node)
		}

		if len(nodes) == 0 {
			u.NextProcedureStep()
			return
		}

		path, err := u.blob.Dijkstra(u.nodeID, RandomSliceElement(nodes).id)
		if err != nil {
			u.SetCurrentProcedureStep(Wander)
			return
		}

		if len(path) == 0 {
			u.NextProcedureStep()
			return
		}

		u.SetTraversalPath(path)

		u.SetCurrentProcedureStep(Traverse)

	case TraverseTo:
		path, err := u.blob.Dijkstra(u.nodeID, u.CurrentProcedureStep().nodeID)
		if err != nil {
			if u.job != nil {
				u.blob.jobs.Halt(u.job)
			}

			u.ClearProcedure()
			u.SetCurrentProcedureStep(Wander)
			return
		}

		if len(path) == 0 {
			u.NextProcedureStep()
			return
		}

		u.SetTraversalPath(path)
		u.SetCurrentProcedureStep(Traverse)

	case Traverse:
		u.traversingProgress += u.conf.TraversalSpeed

		if u.traversingProgress >= u.traversingConnection.Length {
			u.traversingStep++

			u.nodeID = u.traversingConnection.Nodes.Opposite(u.nodeID)
			u.traversingProgress = 0

			if u.traversingStep >= len(u.traversingPath) {
				u.traversingPath = nil
				u.traversingConnection = nil
				u.traversingStep = 0

				u.NextProcedureStep()
				return
			}

			u.traversingConnection = u.blob.GetConnection(
				NewConnectionIDs(u.nodeID, u.traversingPath[u.traversingStep]),
			)
		}
	case StartCarry:
		id, resourceType, err := u.blob.GetProducerNodeID()
		if err != nil {
			u.ClearProcedure()
			u.SetCurrentProcedureStep(Wander)
			return
		}

		u.procedure = []*ProcedureStep{
			{
				stepType: TraverseTo,
				nodeID:   id,
			},
			{
				stepType:     PickUpResource,
				resourceType: resourceType,
			},
			{
				stepType: FindConsumer,
			},
			{
				stepType: DropResource,
			},
		}

	case StartLerp:
		u.stationaryPos = u.blob.Nodes[u.nodeID].pos
		u.stationaryTarget = u.blob.Nodes[u.nodeID].RandPosInNode()
		u.NextProcedureStep()
	case DoJob:
		u.StationaryLerp()

		u.jobProgress += u.job.conf.ProductionSpeed

		if u.jobProgress >= 100 {
			u.jobProgress = 0
			u.blob.jobs.Complete(u.job)

			// find next task or wander
			u.ClearProcedure()
			u.SetCurrentProcedureStep(Wander)
		}
	case PickUpResource:
		err := u.blob.Nodes[u.nodeID].TakeResource(
			u.CurrentProcedureStep().resourceType,
		)
		if err != nil {
			u.ClearProcedure()
			u.SetCurrentProcedureStep(Wander)
			return
		}

		u.resource = u.CurrentProcedureStep().resourceType

		u.NextProcedureStep()

	case FindConsumer:
		consumerNodeID, err := u.blob.GetConsumerNodeID(u.resource)
		if err != nil {
			// TODO: figure out what to do here. carried resource gets lost.
			u.resource = ResourceTypeNone
			u.ClearProcedure()
			u.SetCurrentProcedureStep(Wander)
			return
		}

		path, err := u.blob.Dijkstra(u.nodeID, consumerNodeID)
		if err != nil {
			u.resource = ResourceTypeNone
			u.ClearProcedure()
			u.SetCurrentProcedureStep(Wander)
			return
		}

		if len(path) == 0 {
			u.NextProcedureStep()
			return
		}

		u.SetTraversalPath(path)

		u.SetCurrentProcedureStep(Traverse)
	case DropResource:
		err := u.blob.Nodes[u.nodeID].AddResource(u.resource)
		if err != nil {
			// u.ClearProcedure()
			u.SetCurrentProcedureStep(FindConsumer)
			return
		}

		u.resource = ResourceTypeNone

		u.NextProcedureStep()

	case FindJob:
		job, err := u.blob.jobs.GetJob()
		if err != nil {
			u.SetCurrentProcedureStep(StartCarry)
			return
		}

		if !job.CanDo() {
			u.blob.jobs.Halt(job)
			u.SetCurrentProcedureStep(StartCarry)
			return
		}

		u.job = job

		u.procedure = []*ProcedureStep{
			{
				stepType: TraverseTo,
				nodeID:   job.nodeID,
			},
			{
				stepType: StartLerp,
			},
			{
				stepType: DoJob,
			},
		}
	case FindFood:
		nodeID, _, err := u.blob.GetResourceProducerNodeID(ResourceTypeMushroom)
		if err != nil {
			u.SetCurrentProcedureStep(Wander)
			return
		}

		u.procedure = []*ProcedureStep{
			{
				stepType: TraverseTo,
				nodeID:   nodeID,
			},
			{
				stepType: Eat,
			},
		}
	case Eat:
		err := u.blob.Nodes[u.nodeID].TakeResource(ResourceTypeMushroom)
		if err != nil {
			u.SetCurrentProcedureStep(Wander)
			return
		}
		fmt.Println("eat")

		u.hunger = 0
		u.SetCurrentProcedureStep(FindTask)
	}
}

func (u *Unit) CurrentProcedureStep() *ProcedureStep {
	return u.procedure[0]
}

func (u *Unit) SetCurrentProcedureStep(stepType ProcedureStepType) {
	if len(u.procedure) == 0 {
		u.procedure = make([]*ProcedureStep, 1)
	}

	u.procedure[0] = &ProcedureStep{stepType: stepType}
}

func (u *Unit) NextProcedureStep() {
	// has no procedure or only one step (current one)
	if len(u.procedure) <= 1 {
		u.SetCurrentProcedureStep(FindTask)
		return
	}

	u.procedure = u.procedure[1:]
}

func (u *Unit) ClearProcedure() {
	u.procedure = nil
}

func (u *Unit) StationaryLerp() {
	if u.stationaryLerpProgress >= 1 {
		u.stationaryTarget = u.blob.Nodes[u.nodeID].RandPosInNode()
		u.stationaryLerpProgress = 0
		return
	}

	u.stationaryLerpProgress += 0.05

	u.stationaryPos = pixel.Lerp(
		u.stationaryPos,
		u.stationaryTarget,
		u.stationaryLerpProgress,
	)
}

func (u *Unit) SetTraversalPath(path []int) {
	u.traversingPath = path
	u.traversingConnection = u.blob.GetConnection(
		NewConnectionIDs(u.nodeID, path[0]),
	)
	u.traversingStep = 0
	u.traversingProgress = 0
}

func (u *Unit) Die() {
	fmt.Println("unit died")
	u.blob.jobs.Complete(u.job)

	delete(u.blob.Units, u.id)
}
