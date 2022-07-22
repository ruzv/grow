package blob

import (
	"encoding/json"
	"errors"
	"image/color"
	"math"
	"math/rand"
	"os"

	"private/grow/config"
	"private/grow/render"

	"github.com/faiface/pixel"
)

type Blob struct {
	nodesIdentifier     int
	resourcesIdentifier int
	Nodes               map[int]*Node
	Connections         []*Connection
	Units               []*Unit
	unassignedTasks     *TaskQueue
	connected           map[ConnectionIDs]bool

	conf *config.Config
}

type BlobJSON struct {
	NodesIdentifier     int               `json:"nodes_identifier"`
	ResourcesIdentifier int               `json:"resources_identifier"`
	Nodes               map[int]*NodeJSON `json:"nodes"`
	Connections         []*Connection     `json:"connections"`
	Units               []*UnitJSON       `json:"units"`
	UnassignedTasks     *TaskQueueJSON    `json:"unassigned_tasks"`
}

func NewBlob(bj *BlobJSON, conf *config.Config) *Blob {
	b := &Blob{}

	b.nodesIdentifier = bj.NodesIdentifier
	b.resourcesIdentifier = bj.ResourcesIdentifier

	b.Nodes = make(map[int]*Node)
	for id, node := range bj.Nodes {
		b.Nodes[id] = NewNode(node, b, conf)
	}
	// b.Nodes = bj.Nodes
	b.Connections = bj.Connections

	for _, unit := range bj.Units {
		b.Units = append(b.Units, NewUnit(unit, b))
	}

	b.unassignedTasks = NewTaskQueue(bj.UnassignedTasks)

	b.connected = make(map[ConnectionIDs]bool)
	for _, conn := range bj.Connections {
		b.connected[conn.Nodes] = true
	}

	b.conf = conf

	for _, unit := range b.Units {
		unit.blob = b
	}

	return b
}

func LoadBlob(filepath string, conf *config.Config) (*Blob, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	bj := &BlobJSON{}

	err = json.NewDecoder(f).Decode(&bj)
	if err != nil {
		return nil, err
	}

	return NewBlob(bj, conf), nil
}

func (b *Blob) Save(filepath string) error {
	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		return err
	}

	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "    ")

	return encoder.Encode(b.ToJSON())
}

func (b *Blob) ToJSON() *BlobJSON {
	bj := &BlobJSON{}

	bj.NodesIdentifier = b.nodesIdentifier
	bj.ResourcesIdentifier = b.resourcesIdentifier

	bj.Nodes = make(map[int]*NodeJSON)
	for id, node := range b.Nodes {
		bj.Nodes[id] = node.ToJSON()
	}

	bj.Connections = b.Connections

	bj.Units = make([]*UnitJSON, len(b.Units))
	for i, unit := range b.Units {
		bj.Units[i] = unit.ToJSON()
	}

	// bj.Units = b.Units
	bj.UnassignedTasks = b.unassignedTasks.ToJSON()

	return bj
}

func (b *Blob) String() string {
	s, err := json.MarshalIndent(b, "", "    ")
	if err != nil {
		return err.Error()
	}

	return string(s)
}

func (b *Blob) Render(rend *render.Renderer) {
	for _, conn := range b.Connections {
		n1 := b.Nodes[conn.Nodes.Node1]
		n2 := b.Nodes[conn.Nodes.Node2]

		rend.Line(n1.pos, n2.pos, color.RGBA{255, 255, 255, 255}, 8)
	}

	for _, node := range b.Nodes {
		node.Render(rend)
	}

	for _, unit := range b.Units {
		rend.Circle(unit.Pos(), color.RGBA{50, 50, 50, 255}, 6, 0)
	}
}

func (b *Blob) Update() {
	for _, unit := range b.Units {
		unit.Update()
	}
}

func (b *Blob) AddNode(pos pixel.Vec, nodeType NodeType) int {
	node := NewNode(
		&NodeJSON{
			ID:   b.nodesIdentifier,
			Pos:  pos,
			Type: nodeType,
		},
		b,
		b.conf,
	)

	switch nodeType {
	case NodeTypeMossFarm:
		b.unassignedTasks.Push(&Task{
			taskType: TaskTypeGrowMoss,
			nodeID:   node.id,
		})
		b.unassignedTasks.Push(&Task{
			taskType: TaskTypeGrowMoss,
			nodeID:   node.id,
		})
		b.unassignedTasks.Push(&Task{
			taskType: TaskTypeGrowMoss,
			nodeID:   node.id,
		})
	}

	b.nodesIdentifier++
	b.Nodes[node.id] = node

	return node.id
}

func (b *Blob) AddUnit(nodeID int) *Unit {
	u := &Unit{
		state:  UnitStateStartingWondening,
		nodeID: nodeID,
		blob:   b,
	}

	b.Units = append(b.Units, u)

	return u
}

func (b *Blob) GetConnection(connIDs ConnectionIDs) *Connection {
	for _, conn := range b.Connections {
		if conn.Nodes == connIDs {
			return conn
		}
	}

	return nil
}

func (b *Blob) GetNodeConnections(id int) []*Connection {
	var connections []*Connection

	for _, conn := range b.Connections {
		if conn.Nodes.Node1 == id || conn.Nodes.Node2 == id {
			connections = append(connections, conn)
		}
	}

	return connections
}

func (b *Blob) Connect(id1, id2 int) (*Connection, error) {
	if id1 == id2 {
		return nil, errors.New("cannot connect to self")
	}

	n1 := b.Nodes[id1]
	n2 := b.Nodes[id2]

	if n1 == nil || n2 == nil {
		return nil, errors.New("node not found")
	}

	connIDs := NewConnectionIDs(id1, id2)

	if b.connected[connIDs] {
		return b.GetConnection(connIDs), nil
	}

	return b.addConnection(connIDs), nil
}

func (b *Blob) addConnection(connIDs ConnectionIDs) *Connection {
	n1 := b.Nodes[connIDs.Node1]
	n2 := b.Nodes[connIDs.Node2]

	c := &Connection{
		Nodes:  connIDs,
		Length: n1.pos.Sub(n2.pos).Len(),
	}

	b.connected[connIDs] = true
	b.Connections = append(b.Connections, c)

	return c
}

func (b *Blob) GetClosestNode(pos pixel.Vec) (int, error) {
	dist := math.Inf(1)

	var closestID int
	var found bool

	for _, node := range b.Nodes {
		d := pos.Sub(node.pos).Len()

		if d < dist {
			dist = d
			closestID = node.id
			found = true
		}
	}

	if !found {
		return 0, errors.New("no node found")
	}

	return closestID, nil
}

func (b *Blob) Dijkstra(startNodeID, targetNodeID int) ([]int, error) {
	if startNodeID == targetNodeID {
		return nil, nil
	}

	dist := make(map[int]float64)
	prev := make(map[int]int)

	for _, node := range b.Nodes {
		dist[node.id] = math.Inf(1)
	}

	dist[startNodeID] = 0

	for len(dist) > 0 {
		var nodeID int
		minDist := math.Inf(1)

		// find node id with shortest distance
		for nID, l := range dist {
			if l <= minDist {
				nodeID = nID
				minDist = l
			}
		}

		conns := b.GetNodeConnections(nodeID)
		if len(conns) == 0 {
		}

		for _, conn := range conns {
			neighborID := conn.Nodes.Opposite(nodeID)

			// not in priority queue
			if _, ok := dist[neighborID]; !ok {
				continue
			}

			alt := dist[nodeID] + conn.Length

			if dist[neighborID] > alt {
				dist[neighborID] = alt
				prev[neighborID] = nodeID
			}

		}

		// delete looked at node
		delete(dist, nodeID)
	}

	path := []int{}

	var ok bool
	node := targetNodeID

	for {
		path = append(path, node)

		if node == startNodeID {
			break
		}

		node, ok = prev[node]
		if !ok {
			return nil, errors.New("no path found")
		}
	}

	path = ReverseSlice(path)

	// delete(dist, startNodeID)

	// dist[startNodeID] = 0

	// nodeID := startNodeID

	// conns := b.GetNodeConnections(nodeID)
	// for _, conn := range conns {
	// 	dist[conn.Nodes.Opposite(nodeID)] = conn.Length
	// }

	return path[1:], nil
}

type TaskQueue struct {
	tasks       []*Task
	haltedTasks []*Task
}

type TaskQueueJSON struct {
	Tasks       []*TaskJSON `json:"tasks"`
	HaltedTasks []*TaskJSON `json:"halted_tasks"`
}

func NewTaskQueue(tqj *TaskQueueJSON) *TaskQueue {
	tq := &TaskQueue{}

	if tqj == nil {
		return tq
	}

	for _, task := range tqj.Tasks {
		tq.tasks = append(tq.tasks, NewTask(task))
	}

	for _, task := range tqj.HaltedTasks {
		tq.haltedTasks = append(tq.haltedTasks, NewTask(task))
	}

	return tq
}

func (tq *TaskQueue) ToJSON() *TaskQueueJSON {
	tqj := &TaskQueueJSON{}

	for _, task := range tq.tasks {
		tqj.Tasks = append(tqj.Tasks, task.ToJSON())
	}

	for _, task := range tq.haltedTasks {
		tqj.HaltedTasks = append(tqj.HaltedTasks, task.ToJSON())
	}

	return tqj
}

func (tq *TaskQueue) Push(task *Task) {
	tq.tasks = append(tq.tasks, task)
}

func (tq *TaskQueue) Pop() *Task {
	if len(tq.tasks) == 0 {
		return nil
	}

	task := tq.tasks[0]
	tq.tasks = tq.tasks[1:]

	return task
}

func (tq *TaskQueue) Empty() bool {
	return len(tq.tasks) == 0
}

func (tq *TaskQueue) PushHalted(task *Task) {
	tq.haltedTasks = append(tq.haltedTasks, task)
}

func (tq *TaskQueue) PopHalted() *Task {
	if len(tq.haltedTasks) == 0 {
		return nil
	}

	task := tq.haltedTasks[0]
	tq.haltedTasks = tq.haltedTasks[1:]

	return task
}

func RandomSliceElement[T any](slice []T) T {
	return slice[rand.Intn(len(slice))]
}

func ReverseSlice[T any](slice []T) []T {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
	return slice
}
