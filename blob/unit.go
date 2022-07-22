package blob

import "github.com/faiface/pixel"

type TaskType int

const (
	TaskTypeGrowMoss TaskType = iota
)

type Task struct {
	Type       TaskType `json:"type"`
	NodeID     int      `json:"node_id"`
	Impossible bool     `json:"impossible"`
}

func (t *Task) ToSteps() []TaskStep {
	var steps []TaskStep

	switch t.Type {
	case TaskTypeGrowMoss:
		steps = []TaskStep{
			{
				Type:   TaskStepTypeTraverseTo,
				NodeID: t.NodeID,
			},
			{
				Type:   TaskStepTypeGrowMoss,
				NodeID: t.NodeID,
			},
		}
	}

	return steps
}

type TaskStepType int

const (
	TaskStepTypeTraverseTo TaskStepType = iota
	TaskStepTypeGrowMoss
)

type TaskStep struct {
	Type   TaskStepType `json:"type"`
	NodeID int          `json:"node_id"`
}

type UnitState int

const (
	UnitStateIdle              = iota
	UnitStateStartingWondening // start wondering
	UnitStateTraverseing
	UnitStateFinishedTraversing
	UnitStateGrowingMoss
	UnitStateStartingTask
)

type Unit struct {
	State  UnitState `json:"state"`
	NodeID int       `json:"node"`

	TraversingPath       []int       `json:"traversing_path"`
	TraversingConnection *Connection `json:"traversing_connection"`
	TraversingStep       int         `json:"traversing_step"`
	TraversingProgress   float64     `json:"traversing_progress"`

	Task            *Task      `json:"task"`
	TaskSteps       []TaskStep `json:"task_steps"`
	CurrentTaskStep int        `json:"current_task_step"`

	blob *Blob
}

func (u *Unit) Pos() pixel.Vec {
	switch u.State {
	case UnitStateIdle,
		UnitStateStartingWondening,
		UnitStateFinishedTraversing,
		UnitStateGrowingMoss:

		return u.blob.Nodes[u.NodeID].pos

	case UnitStateTraverseing:
		start := u.blob.Nodes[u.NodeID].pos
		target := u.blob.Nodes[u.TraversingConnection.Nodes.Opposite(u.NodeID)].pos

		return start.Add(target.Sub(start).Scaled(
			u.TraversingProgress / u.TraversingConnection.Length,
		))
	}

	return pixel.V(0, 0)
}

func (u *Unit) Update() {
	switch u.State {
	case UnitStateIdle:
		// do nothing
	case UnitStateStartingWondening:
		var nodes []*Node
		for _, node := range u.blob.Nodes {
			if node.id == u.NodeID {
				continue
			}
			nodes = append(nodes, node)
		}

		path, err := u.blob.Dijkstra(u.NodeID, RandomSliceElement(nodes).id)
		if err != nil {
			u.State = UnitStateStartingWondening
			return
		}

		if len(path) == 0 {
			u.State = UnitStateFinishedTraversing
			return
		}

		u.SetTraversalPath(path)

		u.State = UnitStateTraverseing

	case UnitStateTraverseing:
		u.TraversingProgress += 2

		if u.TraversingProgress >= u.TraversingConnection.Length {
			u.TraversingStep++

			u.NodeID = u.TraversingConnection.Nodes.Opposite(u.NodeID)
			u.TraversingProgress = 0

			if u.TraversingStep >= len(u.TraversingPath) {
				u.TraversingPath = nil
				u.TraversingConnection = nil
				u.TraversingStep = 0

				u.State = UnitStateFinishedTraversing

				return
			}

			u.TraversingConnection = u.blob.GetConnection(
				NewConnectionIDs(u.NodeID, u.TraversingPath[u.TraversingStep]),
			)
		}
	case UnitStateFinishedTraversing:
		u.NextTaskStep()

	case UnitStateStartingTask:
		if u.blob.UnassignedTasks.Empty() {
			u.State = UnitStateStartingWondening
			return
		}

		task := u.blob.UnassignedTasks.Pop()
		if task == nil {
			u.State = UnitStateStartingWondening
			return
		}

		u.Task = task
		u.TaskSteps = task.ToSteps()
		u.CurrentTaskStep = 0

		u.NextTaskStep()
	case UnitStateGrowingMoss:
		// do nothing for now
	}
}

func (u *Unit) ClearTask() {
	u.Task = nil
	u.TaskSteps = nil
	u.CurrentTaskStep = 0
}

func (u *Unit) SetTraversalPath(path []int) {
	u.TraversingPath = path
	u.TraversingConnection = u.blob.GetConnection(
		NewConnectionIDs(u.NodeID, path[0]),
	)
	u.TraversingStep = 0
	u.TraversingProgress = 0
}

func (u *Unit) NextTaskStep() {
	defer func() { u.CurrentTaskStep++ }()

	if u.CurrentTaskStep >= len(u.TaskSteps) {
		u.State = UnitStateStartingTask
		u.ClearTask()
		return
	}

	currentTask := u.TaskSteps[u.CurrentTaskStep]

	switch currentTask.Type {
	case TaskStepTypeTraverseTo:
		path, err := u.blob.Dijkstra(u.NodeID, currentTask.NodeID)
		if err != nil { // path not found
			u.blob.UnassignedTasks.Impossible(u.Task)
			u.ClearTask()
			u.State = UnitStateStartingTask
			return
		}

		if len(path) == 0 { // unit already at node
			u.State = UnitStateStartingWondening
			return
		}

		u.SetTraversalPath(path)
		u.State = UnitStateTraverseing

	case TaskStepTypeGrowMoss:
		u.State = UnitStateGrowingMoss
	}
}
