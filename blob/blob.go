package blob

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"os"

	"private/grow/render"

	"github.com/faiface/pixel"
)

type BlobConfig struct {
	Nodes map[NodeType]*NodeConfig `json:"nodes"`
	Jobs  map[JobType]*JobConfig   `json:"jobs"`
}

type Blob struct {
	nodesIdentifier     int
	resourcesIdentifier int
	jobsIdentifier      int
	Nodes               map[int]*Node
	Connections         []*Connection // TODO: make this a map[ConnectionIDs]*connection
	Units               []*Unit
	jobs                *JobQueue
	consumers           map[ResourceType][]int // mapping from resource type to node IDs
	producers           map[ResourceType][]int // mapping from resource type to node IDs
	pathCache           map[string][]int       // TODO: make this a map[ConnectionIDs][]int
	connected           map[ConnectionIDs]bool

	conf *BlobConfig
}

type BlobJSON struct {
	NodesIdentifier     int                    `json:"nodes_identifier"`
	ResourcesIdentifier int                    `json:"resources_identifier"`
	JobsIdentifier      int                    `json:"jobs_identifier"`
	Nodes               map[int]*NodeJSON      `json:"nodes"`
	Connections         []*Connection          `json:"connections"`
	Units               []*UnitJSON            `json:"units"`
	Jobs                *JobQueueJSON          `json:"jobs"`
	Consumers           map[ResourceType][]int `json:"consumers"`
	Producers           map[ResourceType][]int `json:"producers"`

	// TODO: SAVE PATH CACHE
	// PathCache           map[ConnectionIDs][]int `json:"path_cache"`
}

func NewBlob(bj *BlobJSON, conf *BlobConfig) *Blob {
	b := &Blob{
		nodesIdentifier:     bj.NodesIdentifier,
		resourcesIdentifier: bj.ResourcesIdentifier,
		jobsIdentifier:      bj.JobsIdentifier,
		Nodes:               make(map[int]*Node),
		Connections:         bj.Connections,
		Units:               make([]*Unit, 0, len(bj.Units)),
		pathCache:           make(map[string][]int),
		connected:           make(map[ConnectionIDs]bool),
		conf:                conf,
	}

	for id, node := range bj.Nodes {
		b.Nodes[id] = NewNode(node, b, conf)
	}

	for _, unit := range bj.Units {
		b.Units = append(b.Units, NewUnit(unit, b))
	}

	for _, conn := range bj.Connections {
		b.connected[conn.Nodes] = true
	}

	b.consumers = bj.Consumers
	if b.consumers == nil {
		b.consumers = make(map[ResourceType][]int)
	}

	b.producers = bj.Producers
	if b.producers == nil {
		b.producers = make(map[ResourceType][]int)
	}

	b.jobs = NewJobQueue(bj.Jobs, conf, b)

	return b
}

func (b *Blob) ToJSON() *BlobJSON {
	bj := &BlobJSON{}

	bj.NodesIdentifier = b.nodesIdentifier
	bj.ResourcesIdentifier = b.resourcesIdentifier
	bj.JobsIdentifier = b.jobsIdentifier

	bj.Nodes = make(map[int]*NodeJSON)
	for id, node := range b.Nodes {
		bj.Nodes[id] = node.ToJSON()
	}

	bj.Connections = b.Connections

	bj.Units = make([]*UnitJSON, len(b.Units))
	for i, unit := range b.Units {
		bj.Units[i] = unit.ToJSON()
	}

	bj.Jobs = b.jobs.ToJSON()

	bj.Consumers = b.consumers
	bj.Producers = b.producers

	return bj
}

func LoadBlob(filepath string, conf *BlobConfig) (*Blob, error) {
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
		unit.Render(rend)
	}
}

func (b *Blob) Update() {
	for _, unit := range b.Units {
		unit.Update()
	}

	for _, node := range b.Nodes {
		node.Update()
	}
}

func (b *Blob) AddNode(pos pixel.Vec, nodeType NodeType) int {
	node := NewNode(
		&NodeJSON{
			ID:       b.nodesIdentifier,
			Pos:      pos,
			NodeType: nodeType,
		},
		b,
		b.conf,
	)

	for _, res := range node.conf.Consumes {
		b.consumers[res] = append(b.consumers[res], node.id)
	}

	for _, res := range node.conf.Produces {
		b.producers[res] = append(b.producers[res], node.id)
	}

	b.jobs.Add(node.Jobs()...)

	b.nodesIdentifier++
	b.Nodes[node.id] = node

	return node.id
}

func (b *Blob) AddUnit(nodeID int) *Unit {
	u := &Unit{
		// TODO: change to pretty method call when it is implemented
		procedure: []*ProcedureStep{{stepType: StartWondening}},
		nodeID:    nodeID,
		blob:      b,
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

	pathID := fmt.Sprintf("%d-%d", startNodeID, targetNodeID)

	path, ok := b.pathCache[pathID]
	if ok {
		return path, nil
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

	path = []int{}
	ok = false

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

	path = ReverseSlice(path)[1:]

	b.pathCache[pathID] = path

	return path, nil
}

func (b *Blob) FindConsumerNodeID(resourceType ResourceType) (int, error) {
	ids, ok := b.consumers[resourceType]
	if !ok {
		return 0, errors.New("no consumer found")
	}

	if len(ids) == 0 {
		return 0, errors.New("no consumer found")
	}

	var (
		max   int
		minID int
		found bool
	)

	for _, id := range ids {
		c := b.Nodes[id].AvailableCapacity(resourceType)
		if c > max {
			max = c
			minID = id
			found = true
		}
	}

	if !found {
		return 0, errors.New("no consumer found")
	}

	return minID, nil
}

func (b *Blob) FindProducerNodeID() (int, ResourceType, error) {
	for resourceType, ids := range b.producers {
		for _, id := range ids {
			if b.Nodes[id].Resources(resourceType) > 0 {
				return id, resourceType, nil
			}
		}
	}

	return 0, "", errors.New("no producer found")
}

type JobQueue struct {
	occupied  map[int]*Job
	available map[int]*Job
	halted    map[int]*Job
}

type JobQueueJSON struct {
	Occupied  map[int]*JobJSON `json:"occupied"`
	Available map[int]*JobJSON `json:"available"`
	Halted    map[int]*JobJSON `json:"halted"`
}

func NewJobQueue(jqj *JobQueueJSON, conf *BlobConfig, blob *Blob) *JobQueue {
	jq := &JobQueue{
		occupied:  make(map[int]*Job),
		available: make(map[int]*Job),
		halted:    make(map[int]*Job),
	}

	if jqj == nil {
		return jq
	}

	for id, job := range jqj.Occupied {
		jq.occupied[id] = NewJob(job, conf, blob)
	}

	for id, job := range jqj.Available {
		jq.available[id] = NewJob(job, conf, blob)
	}

	for id, job := range jqj.Halted {
		jq.halted[id] = NewJob(job, conf, blob)
	}

	return jq
}

func (jq *JobQueue) ToJSON() *JobQueueJSON {
	jqj := &JobQueueJSON{
		Occupied:  make(map[int]*JobJSON),
		Available: make(map[int]*JobJSON),
		Halted:    make(map[int]*JobJSON),
	}

	for id, job := range jq.occupied {
		jqj.Occupied[id] = job.ToJSON()
	}

	for id, job := range jq.available {
		jqj.Available[id] = job.ToJSON()
	}

	for id, job := range jq.halted {
		jqj.Halted[id] = job.ToJSON()
	}

	return jqj
}

func (jq *JobQueue) GetJob() (*Job, error) {
	job, err := jq.getAvailable()
	if err != nil {
		job, err = jq.getHalted()
		if err != nil {
			return nil, err
		}
	}

	jq.occupied[job.id] = job

	return job, nil
}

func (jq *JobQueue) getAvailable() (*Job, error) {
	if len(jq.available) == 0 {
		return nil, errors.New("no available job")
	}

	var job *Job

	for _, j := range jq.available {
		job = j
		break
	}

	delete(jq.available, job.id)

	return job, nil
}

func (jq *JobQueue) getHalted() (*Job, error) {
	if len(jq.halted) == 0 {
		return nil, errors.New("no halted job")
	}

	var job *Job

	for _, j := range jq.halted {
		job = j
		break
	}

	delete(jq.halted, job.id)

	return job, nil
}

func (jq *JobQueue) Complete(job *Job) {
	if job == nil {
		fmt.Println("attempt to complete nil job")
		return
	}

	defer func() { delete(jq.occupied, job.id) }()

	err := job.Complete()
	if err != nil {
		jq.halted[job.id] = job
		return
	}

	jq.available[job.id] = job
}

func (jq *JobQueue) Add(jobs ...*Job) {
	for _, job := range jobs {
		jq.available[job.id] = job
	}
}

func (jq *JobQueue) Halt(job *Job) {
	if job == nil {
		fmt.Println("attempt to halt nil job")
		return
	}

	_, ok := jq.occupied[job.id]
	if !ok {
		fmt.Println("attempt to halt job not in queue")
		return
	}

	delete(jq.occupied, job.id)

	jq.halted[job.id] = job
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
