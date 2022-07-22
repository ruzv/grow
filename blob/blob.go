package blob

import (
	"encoding/json"
	"errors"
	"image/color"
	"math"
	"math/rand"
	"os"

	"private/grow/render"

	"github.com/faiface/pixel"
)

type Blob struct {
	Nodes           map[int]*Node `json:"nodes"`
	NodesIdentifier int           `json:"nodes_identifier"`
	Connections     []*Connection `json:"connections"`
	Units           []*Unit       `json:"units"`
	UnassignedTasks TaskQueue     `json:"unassigned_tasks"`
	connected       map[ConnectionIDs]bool
}

type BlobJSON struct {
	Nodes           map[int]*NodeJSON `json:"nodes"`
	NodesIdentifier int               `json:"nodes_identifier"`
	Connections     []*Connection     `json:"connections"`
	Units           []*Unit           `json:"units"`
	UnassignedTasks TaskQueue         `json:"unassigned_tasks"`
}

func NewBlob(bj *BlobJSON) *Blob {
	b := &Blob{}

	b.Nodes = make(map[int]*Node)
	for id, node := range bj.Nodes {
		b.Nodes[id] = NewNode(node)
	}
	// b.Nodes = bj.Nodes
	b.NodesIdentifier = bj.NodesIdentifier
	b.Connections = bj.Connections
	b.Units = bj.Units
	b.UnassignedTasks = bj.UnassignedTasks

	b.connected = make(map[ConnectionIDs]bool)
	for _, conn := range bj.Connections {
		b.connected[conn.Nodes] = true
	}

	for _, unit := range b.Units {
		unit.blob = b
	}

	return b
}

func LoadBlob(filepath string) (*Blob, error) {
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

	return NewBlob(bj), nil
}

func (b *Blob) Save(filepath string) error {
	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		return err
	}

	defer f.Close()

	return json.NewEncoder(f).Encode(b)
}

func (b *Blob) ToJSON() *BlobJSON {
	bj := &BlobJSON{}

	bj.Nodes = make(map[int]*NodeJSON)
	for id, node := range b.Nodes {
		bj.Nodes[id] = node.ToJSON()
	}

	bj.NodesIdentifier = b.NodesIdentifier
	bj.Connections = b.Connections
	bj.Units = b.Units
	bj.UnassignedTasks = b.UnassignedTasks

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
	node := &Node{
		id:       b.NodesIdentifier,
		pos:      pos,
		nodeType: nodeType,
	}

	switch nodeType {
	case NodeTypeMossFarm:
		b.UnassignedTasks.Push(&Task{
			Type:   TaskTypeGrowMoss,
			NodeID: node.id,
		})
		b.UnassignedTasks.Push(&Task{
			Type:   TaskTypeGrowMoss,
			NodeID: node.id,
		})
		b.UnassignedTasks.Push(&Task{
			Type:   TaskTypeGrowMoss,
			NodeID: node.id,
		})
	}

	b.NodesIdentifier++
	b.Nodes[node.id] = node

	return node.id
}

func (b *Blob) AddUnit(nodeID int) *Unit {
	u := &Unit{
		State:  UnitStateStartingWondening,
		NodeID: nodeID,
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
	Tasks           []*Task `json:"tasks"`
	ImpossibleTasks []*Task `json:"impossible_tasks"`
}

func (tq *TaskQueue) Push(task *Task) {
	tq.Tasks = append(tq.Tasks, task)
}

func (tq *TaskQueue) Pop() *Task {
	if len(tq.Tasks) == 0 {
		return nil
	}

	task := tq.Tasks[0]
	tq.Tasks = tq.Tasks[1:]

	return task
}

func (tq *TaskQueue) Empty() bool {
	return len(tq.Tasks) == 0
}

func (tq *TaskQueue) Impossible(task *Task) {
	tq.ImpossibleTasks = append(tq.ImpossibleTasks, task)
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
