package blob

import (
	"fmt"
	"image/color"

	"private/grow/render"

	"github.com/faiface/pixel"
)

// action - single step in units procedure
// procedure - sequence of procedureSteps that the unit tries to do
// procedureStep - groups together the action and the reqired meta-data for that
// action

type ProcedureStepType string // ProcedureStepType

const (
	DoNothing         ProcedureStepType = "do_nothing"
	StartWondening    ProcedureStepType = "start_wondening"
	FindTraversalPath ProcedureStepType = "find_traversal_path" // rename TraverseTo
	Traverse          ProcedureStepType = "traverse"
	FinishTraversing  ProcedureStepType = "finish_traversing"
	FindTask          ProcedureStepType = "find_task"
	StartLerp         ProcedureStepType = "start_lerp"
	PickUpResource    ProcedureStepType = "pick_up_resource"
	DropResource      ProcedureStepType = "drop_resource"
	FindConsumer      ProcedureStepType = "find_consumer"
	FindJob           ProcedureStepType = "find_job"
	DoJob             ProcedureStepType = "do_job"
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

// units should try to satisfy the needs of consummers

type Unit struct {
	procedure []*ProcedureStep
	nodeID    int

	TraversingPath       []int
	TraversingConnection *Connection
	TraversingStep       int
	TraversingProgress   float64

	stationaryPos          pixel.Vec
	stationaryTarget       pixel.Vec
	stationaryLerpProgress float64

	resource ResourceType

	job         *Job
	jobProgress float64

	blob *Blob
}

type UnitJSON struct {
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
}

func NewUnit(uj *UnitJSON, blob *Blob) *Unit {
	u := &Unit{
		procedure: make([]*ProcedureStep, 0, len(uj.Procedure)),
		nodeID:    uj.NodeID,

		TraversingPath:       uj.TraversingPath,
		TraversingConnection: uj.TraversingConnection,
		TraversingStep:       uj.TraversingStep,
		TraversingProgress:   uj.TraversingProgress,

		stationaryPos:          uj.StationaryPos,
		stationaryTarget:       uj.StationaryTarget,
		stationaryLerpProgress: uj.StationaryLerpProgress,

		resource: uj.Resource,

		job:         NewJob(uj.Job, blob.conf, blob),
		jobProgress: uj.JobProgress,

		blob: blob,
	}

	for _, p := range uj.Procedure {
		u.procedure = append(u.procedure, NewProcedureStep(p))
	}

	return u
}

func (u *Unit) ToJSON() *UnitJSON {
	uj := &UnitJSON{
		Procedure: make([]*ProcedureStepJSON, 0, len(u.procedure)),
		NodeID:    u.nodeID,

		TraversingPath:       u.TraversingPath,
		TraversingConnection: u.TraversingConnection,
		TraversingStep:       u.TraversingStep,
		TraversingProgress:   u.TraversingProgress,

		StationaryPos:          u.stationaryPos,
		StationaryTarget:       u.stationaryTarget,
		StationaryLerpProgress: u.stationaryLerpProgress,

		Resource: u.resource,

		Job:         u.job.ToJSON(),
		JobProgress: u.jobProgress,
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
		target := u.blob.Nodes[u.TraversingConnection.Nodes.Opposite(u.nodeID)].pos

		return start.Add(target.Sub(start).Scaled(
			u.TraversingProgress / u.TraversingConnection.Length,
		))
	}

	return u.blob.Nodes[u.nodeID].pos
}

func (u *Unit) Update() {
	switch u.CurrentProcedureStep().stepType {
	case DoNothing:
		// do nothing
	case StartWondening:
		var nodes []*Node
		for _, node := range u.blob.Nodes {
			if node.id == u.nodeID {
				continue
			}
			nodes = append(nodes, node)
		}

		if len(nodes) == 0 {
			u.SetCurrentProcedureStep(FinishTraversing)
			return
		}

		path, err := u.blob.Dijkstra(u.nodeID, RandomSliceElement(nodes).id)
		if err != nil {
			u.SetCurrentProcedureStep(StartWondening)
			return
		}

		if len(path) == 0 {
			u.SetCurrentProcedureStep(FinishTraversing)
			return
		}

		u.SetTraversalPath(path)

		u.SetCurrentProcedureStep(Traverse)

	case FindTraversalPath:
		path, err := u.blob.Dijkstra(u.nodeID, u.CurrentProcedureStep().nodeID)
		if err != nil {
			if u.job != nil {
				u.blob.jobs.Halt(u.job)
			}

			u.ClearProcedure()
			u.SetCurrentProcedureStep(StartWondening)
			return
		}

		if len(path) == 0 {
			u.SetCurrentProcedureStep(FinishTraversing)
			return
		}

		u.SetTraversalPath(path)
		u.SetCurrentProcedureStep(Traverse)

	case Traverse:
		u.TraversingProgress += 2

		if u.TraversingProgress >= u.TraversingConnection.Length {
			u.TraversingStep++

			u.nodeID = u.TraversingConnection.Nodes.Opposite(u.nodeID)
			u.TraversingProgress = 0

			if u.TraversingStep >= len(u.TraversingPath) {
				u.TraversingPath = nil
				u.TraversingConnection = nil
				u.TraversingStep = 0

				u.SetCurrentProcedureStep(FinishTraversing)

				return
			}

			u.TraversingConnection = u.blob.GetConnection(
				NewConnectionIDs(u.nodeID, u.TraversingPath[u.TraversingStep]),
			)
		}
	case FinishTraversing:
		u.NextProcedureStep()
	case FindTask:
		id, resourceType, err := u.blob.FindProducerNodeID()
		if err != nil {
			fmt.Println(err)
			u.ClearProcedure()
			u.SetCurrentProcedureStep(StartWondening)
			return
		}

		u.procedure = []*ProcedureStep{
			{
				stepType: FindTraversalPath,
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
		// TODO: implement job.CanDo() method that would check if all the
		// required resources on node. This propably has to be done in procedure
		// step StartDoingJob

		u.StationaryLerp()

		u.jobProgress += u.job.conf.ProductionSpeed

		if u.jobProgress >= 100 {
			u.jobProgress = 0
			u.blob.jobs.Complete(u.job)

			// find next task or wander
			u.ClearProcedure()
			u.SetCurrentProcedureStep(StartWondening)
		}
	case PickUpResource:
		err := u.blob.Nodes[u.nodeID].TakeResource(
			u.CurrentProcedureStep().resourceType,
		)
		if err != nil {
			u.ClearProcedure()
			u.SetCurrentProcedureStep(StartWondening)
			return
		}

		u.resource = u.CurrentProcedureStep().resourceType

		u.NextProcedureStep()

	case FindConsumer:
		consumerNodeID, err := u.blob.FindConsumerNodeID(u.resource)
		if err != nil {
			// TODO: figure out what to do here. carried resource gets lost.
			u.resource = ResourceTypeNone
			u.ClearProcedure()
			u.SetCurrentProcedureStep(StartWondening)
			return
		}

		path, err := u.blob.Dijkstra(u.nodeID, consumerNodeID)
		if err != nil {
			u.resource = ResourceTypeNone
			u.ClearProcedure()
			u.SetCurrentProcedureStep(StartWondening)
			return
		}

		if len(path) == 0 {
			u.SetCurrentProcedureStep(FinishTraversing)
			return
		}

		u.SetTraversalPath(path)

		u.SetCurrentProcedureStep(Traverse)
	case DropResource:
		err := u.blob.Nodes[u.nodeID].AddResource(u.resource)
		if err != nil {
			u.ClearProcedure()
			u.SetCurrentProcedureStep(StartWondening)
			return
		}

		u.resource = ResourceTypeNone

		u.NextProcedureStep()

	case FindJob:
		job, err := u.blob.jobs.GetJob()
		if err != nil {
			u.SetCurrentProcedureStep(FindTask)
			return
		}

		u.job = job

		u.procedure = []*ProcedureStep{
			{
				stepType: FindTraversalPath,
				nodeID:   job.nodeID,
			},
			{
				stepType: StartLerp,
			},
			{
				stepType: DoJob,
			},
		}
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
		u.SetCurrentProcedureStep(FindJob)
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
	u.TraversingPath = path
	u.TraversingConnection = u.blob.GetConnection(
		NewConnectionIDs(u.nodeID, path[0]),
	)
	u.TraversingStep = 0
	u.TraversingProgress = 0
}
