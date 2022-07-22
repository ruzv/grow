package blob

import (
	"fmt"

	"github.com/faiface/pixel"
)

type TaskType string

const (
	TaskTypeGrowMoss TaskType = "grow_moss"
	TaskTypeCarry    TaskType = "carry"
)

type Task struct {
	taskType TaskType
	nodeID   int
}

type TaskJSON struct {
	Type   TaskType `json:"type"`
	NodeID int      `json:"node_id"`
}

func NewTask(tj *TaskJSON) *Task {
	if tj == nil {
		return nil
	}

	return &Task{
		taskType: tj.Type,
		nodeID:   tj.NodeID,
	}
}

func (t *Task) ToJSON() *TaskJSON {
	if t == nil {
		return nil
	}

	return &TaskJSON{
		Type:   t.taskType,
		NodeID: t.nodeID,
	}
}

func (t *Task) Steps() []TaskStep {
	var steps []TaskStep

	switch t.taskType {
	case TaskTypeGrowMoss:
		steps = []TaskStep{
			{
				Type:   TaskStepTypeTraverseTo,
				NodeID: t.nodeID,
			},
			{
				Type:   TaskStepStartLerp,
				NodeID: t.nodeID,
			},
			{
				Type:   TaskStepTypeGrowMoss,
				NodeID: t.nodeID,
			},
		}
	}

	return steps
}

type TaskStepType string

const (
	TaskStepTypeTraverseTo TaskStepType = "traverse_to"
	TaskStepTypeGrowMoss   TaskStepType = "grow_moss"
	TaskStepStartLerp      TaskStepType = "start_lerp"
)

type TaskStep struct {
	Type   TaskStepType `json:"type"`
	NodeID int          `json:"node_id"`
}

type UnitState string

const (
	UnitStateIdle               UnitState = "idle"
	UnitStateStartingWondening  UnitState = "starting_wondening"
	UnitStateTraverseing        UnitState = "traverseing"
	UnitStateFinishedTraversing UnitState = "finished_traversing"
	UnitStateGrowingMoss        UnitState = "growing_moss"
	UnitStateStartingTask       UnitState = "starting_task"
	UnitStateSartingLerp        UnitState = "starting_lerp"
)

type Unit struct {
	state  UnitState
	nodeID int

	TraversingPath       []int
	TraversingConnection *Connection
	TraversingStep       int
	TraversingProgress   float64

	Task            *Task
	TaskSteps       []TaskStep
	CurrentTaskStep int

	stationaryPos          pixel.Vec
	stationaryTarget       pixel.Vec
	stationaryLerpProgress float64

	productionProgress float64

	blob *Blob
}

type UnitJSON struct {
	State  UnitState `json:"state"`
	NodeID int       `json:"node"`

	TraversingPath       []int       `json:"traversing_path"`
	TraversingConnection *Connection `json:"traversing_connection"`
	TraversingStep       int         `json:"traversing_step"`
	TraversingProgress   float64     `json:"traversing_progress"`

	Task            *TaskJSON  `json:"task"`
	TaskSteps       []TaskStep `json:"task_steps"`
	CurrentTaskStep int        `json:"current_task_step"`

	StationaryPos          pixel.Vec `json:"stationary_pos"`
	StationaryTarget       pixel.Vec `json:"stationary_target"`
	StationaryLerpProgress float64   `json:"stationary_lerp_progress"`

	ProductionProgress float64 `json:"production_progress"`
}

func NewUnit(uj *UnitJSON, b *Blob) *Unit {
	u := &Unit{
		state:  uj.State,
		nodeID: uj.NodeID,

		TraversingPath:       uj.TraversingPath,
		TraversingConnection: uj.TraversingConnection,
		TraversingStep:       uj.TraversingStep,
		TraversingProgress:   uj.TraversingProgress,

		Task:            NewTask(uj.Task),
		TaskSteps:       uj.TaskSteps,
		CurrentTaskStep: uj.CurrentTaskStep,

		stationaryPos:          uj.StationaryPos,
		stationaryTarget:       uj.StationaryTarget,
		stationaryLerpProgress: uj.StationaryLerpProgress,

		productionProgress: uj.ProductionProgress,

		blob: b,
	}

	return u
}

func (u *Unit) ToJSON() *UnitJSON {
	return &UnitJSON{
		State:  u.state,
		NodeID: u.nodeID,

		TraversingPath:       u.TraversingPath,
		TraversingConnection: u.TraversingConnection,
		TraversingStep:       u.TraversingStep,
		TraversingProgress:   u.TraversingProgress,

		Task:            u.Task.ToJSON(),
		TaskSteps:       u.TaskSteps,
		CurrentTaskStep: u.CurrentTaskStep,

		StationaryPos:          u.stationaryPos,
		StationaryTarget:       u.stationaryTarget,
		StationaryLerpProgress: u.stationaryLerpProgress,

		ProductionProgress: u.productionProgress,
	}
}

func (u *Unit) Pos() pixel.Vec {
	switch u.state {
	case UnitStateIdle,
		UnitStateStartingWondening,
		UnitStateFinishedTraversing,
		UnitStateSartingLerp,
		UnitStateStartingTask:

		return u.blob.Nodes[u.nodeID].pos

	case UnitStateGrowingMoss:
		return u.stationaryPos

	case UnitStateTraverseing:
		start := u.blob.Nodes[u.nodeID].pos
		target := u.blob.Nodes[u.TraversingConnection.Nodes.Opposite(u.nodeID)].pos

		return start.Add(target.Sub(start).Scaled(
			u.TraversingProgress / u.TraversingConnection.Length,
		))
	}

	fmt.Println("missing position for ", u.state)
	return pixel.V(0, 0)
}

func (u *Unit) Update() {
	switch u.state {
	case UnitStateIdle:
		// do nothing
	case UnitStateStartingWondening:
		var nodes []*Node
		for _, node := range u.blob.Nodes {
			if node.id == u.nodeID {
				continue
			}
			nodes = append(nodes, node)
		}

		if len(nodes) == 0 {
			u.state = UnitStateFinishedTraversing
			return
		}

		path, err := u.blob.Dijkstra(u.nodeID, RandomSliceElement(nodes).id)
		if err != nil {
			u.state = UnitStateStartingWondening
			return
		}

		if len(path) == 0 {
			u.state = UnitStateFinishedTraversing
			return
		}

		u.SetTraversalPath(path)

		u.state = UnitStateTraverseing

	case UnitStateTraverseing:
		u.TraversingProgress += 2

		if u.TraversingProgress >= u.TraversingConnection.Length {
			u.TraversingStep++

			u.nodeID = u.TraversingConnection.Nodes.Opposite(u.nodeID)
			u.TraversingProgress = 0

			if u.TraversingStep >= len(u.TraversingPath) {
				u.TraversingPath = nil
				u.TraversingConnection = nil
				u.TraversingStep = 0

				u.state = UnitStateFinishedTraversing

				return
			}

			u.TraversingConnection = u.blob.GetConnection(
				NewConnectionIDs(u.nodeID, u.TraversingPath[u.TraversingStep]),
			)
		}
	case UnitStateFinishedTraversing:
		u.NextTaskStep()

	case UnitStateStartingTask:

		// var task *Task

		// if u.blob.unassignedTasks.Empty() {
		// 	task

		// 	u.state = UnitStateStartingWondening
		// 	return
		// }

		task := u.blob.unassignedTasks.Pop()
		if task == nil {
			task = u.blob.unassignedTasks.PopHalted()
			if task == nil {
				u.state = UnitStateStartingWondening
				return
			}
		}

		u.Task = task
		u.TaskSteps = task.Steps()
		u.CurrentTaskStep = 0

		u.NextTaskStep()

	case UnitStateSartingLerp:
		u.stationaryPos = u.blob.Nodes[u.nodeID].pos
		u.stationaryTarget = u.blob.Nodes[u.nodeID].RandPosInNode()
		u.NextTaskStep()

	case UnitStateGrowingMoss:
		u.StationaryLerp()

		u.productionProgress += 0.1

		if u.productionProgress >= 20 {
			u.productionProgress = 0
			_, err := u.blob.Nodes[u.nodeID].AddResource(ResourceTypeMoss)
			if err != nil { // node full
				u.blob.unassignedTasks.PushHalted(u.Task)
				u.ClearTask()
				u.state = UnitStateStartingWondening

				return
			}

		}

		// do nothing for now
	}
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

func (u *Unit) ClearTask() {
	u.Task = nil
	u.TaskSteps = nil
	u.CurrentTaskStep = 0
}

func (u *Unit) SetTraversalPath(path []int) {
	u.TraversingPath = path
	u.TraversingConnection = u.blob.GetConnection(
		NewConnectionIDs(u.nodeID, path[0]),
	)
	u.TraversingStep = 0
	u.TraversingProgress = 0
}

func (u *Unit) NextTaskStep() {
	defer func() { u.CurrentTaskStep++ }()

	if u.CurrentTaskStep >= len(u.TaskSteps) {
		u.state = UnitStateStartingTask
		u.ClearTask()
		return
	}

	currentTask := u.TaskSteps[u.CurrentTaskStep]

	switch currentTask.Type {
	case TaskStepTypeTraverseTo:
		path, err := u.blob.Dijkstra(u.nodeID, currentTask.NodeID)
		fmt.Println(path)
		if err != nil { // path not found
			u.blob.unassignedTasks.PushHalted(u.Task)
			u.ClearTask()
			u.state = UnitStateStartingTask
			return
		}

		if len(path) == 0 { // unit already at node
			u.state = UnitStateFinishedTraversing
			return
		}

		u.SetTraversalPath(path)
		u.state = UnitStateTraverseing
	case TaskStepStartLerp:
		u.state = UnitStateSartingLerp
	case TaskStepTypeGrowMoss:
		u.state = UnitStateGrowingMoss
	}
}
