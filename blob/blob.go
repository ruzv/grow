package blob

import (
	"errors"
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"private/grow/render"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
)

type BlobConfig struct {
	Nodes     map[NodeType]*NodeConfig         `json:"nodes"`
	Jobs      map[JobType]*JobConfig           `json:"jobs"`
	Resources map[ResourceType]*ResourceConfig `json:"resources"`
	Unit      *UnitConfig                      `json:"unit"`
}

type Blob struct {
	nodesIdentifier     int
	resourcesIdentifier int
	jobsIdentifier      int
	unitsIdentifier     int
	Nodes               map[int]*Node
	Connections         []*Connection // TODO: make this a map[ConnectionIDs]*connection
	Units               map[int]*Unit
	jobs                *JobQueue
	consumers           map[ResourceType]map[int][]int // mapping from resource type to map of priorities to node IDs
	producers           map[ResourceType]map[int][]int // mapping from resource type to node IDs
	pathCache           map[string][]int               // TODO: make this a map[ConnectionIDs][]int
	connected           map[ConnectionIDs]bool

	rend *render.Renderer
	conf *BlobConfig
}

type BlobJSON struct {
	NodesIdentifier     int                            `json:"nodes_identifier"`
	ResourcesIdentifier int                            `json:"resources_identifier"`
	JobsIdentifier      int                            `json:"jobs_identifier"`
	UnitsIdentifier     int                            `json:"units_identifier"`
	Nodes               map[int]*NodeJSON              `json:"nodes"`
	Connections         []*Connection                  `json:"connections"`
	Units               map[int]*UnitJSON              `json:"units"`
	Jobs                *JobQueueJSON                  `json:"jobs"`
	Consumers           map[ResourceType]map[int][]int `json:"consumers"`
	Producers           map[ResourceType]map[int][]int `json:"producers"`

	// TODO: SAVE PATH CACHE
	// PathCache           map[ConnectionIDs][]int `json:"path_cache"`
}

func NewBlob(bj *BlobJSON, conf *BlobConfig, win *pixelgl.Window) *Blob {
	b := &Blob{
		nodesIdentifier:     bj.NodesIdentifier,
		resourcesIdentifier: bj.ResourcesIdentifier,
		jobsIdentifier:      bj.JobsIdentifier,
		unitsIdentifier:     bj.UnitsIdentifier,
		Nodes:               make(map[int]*Node),
		Connections:         bj.Connections,
		Units:               make(map[int]*Unit),
		pathCache:           make(map[string][]int),
		connected:           make(map[ConnectionIDs]bool),
		rend:                render.NewRenderer(win),
		conf:                conf,
	}

	for id, node := range bj.Nodes {
		b.Nodes[id] = NewNode(node, b, conf)
	}

	for _, unit := range bj.Units {
		b.Units[unit.ID] = NewUnit(unit, b)
	}

	for _, conn := range bj.Connections {
		b.connected[conn.Nodes] = true
	}

	b.consumers = bj.Consumers
	if b.consumers == nil {
		b.consumers = make(map[ResourceType]map[int][]int)
	}

	b.producers = bj.Producers
	if b.producers == nil {
		b.producers = make(map[ResourceType]map[int][]int)
	}

	b.jobs = NewJobQueue(bj.Jobs, conf, b)

	return b
}

func (b *Blob) ToJSON() *BlobJSON {
	bj := &BlobJSON{}

	bj.NodesIdentifier = b.nodesIdentifier
	bj.ResourcesIdentifier = b.resourcesIdentifier
	bj.JobsIdentifier = b.jobsIdentifier
	bj.UnitsIdentifier = b.unitsIdentifier

	bj.Nodes = make(map[int]*NodeJSON)
	for id, node := range b.Nodes {
		bj.Nodes[id] = node.ToJSON()
	}

	bj.Connections = b.Connections

	bj.Units = make(map[int]*UnitJSON)
	for id, unit := range b.Units {
		bj.Units[id] = unit.ToJSON()
	}

	bj.Jobs = b.jobs.ToJSON()

	bj.Consumers = b.consumers
	bj.Producers = b.producers

	return bj
}

func (b *Blob) Render() {
	for _, conn := range b.Connections {
		n1 := b.Nodes[conn.Nodes.Node1]
		n2 := b.Nodes[conn.Nodes.Node2]

		b.rend.Line(n1.pos, n2.pos, color.RGBA{255, 255, 255, 255}, 8)
	}

	for _, node := range b.Nodes {
		node.Render(b.rend)
	}

	for _, unit := range b.Units {
		unit.Render(b.rend)
	}

	b.rend.Render()
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

	for res, priority := range node.conf.Consumes {
		priorities := b.consumers[res]
		if priorities == nil {
			priorities = make(map[int][]int)
			b.consumers[res] = priorities
		}

		priorities[priority] = append(b.consumers[res][priority], node.id)
	}

	for res, priority := range node.conf.Produces {
		priorities := b.producers[res]
		if priorities == nil {
			priorities = make(map[int][]int)
			b.producers[res] = priorities
		}

		priorities[priority] = append(priorities[priority], node.id)
	}

	b.jobs.Add(node.Jobs()...)

	b.nodesIdentifier++
	b.Nodes[node.id] = node

	return node.id
}

func (b *Blob) AddUnit(nodeID int) *Unit {
	u := NewUnit(&UnitJSON{NodeID: nodeID}, b)
	u.id = b.unitsIdentifier
	b.unitsIdentifier++
	u.SetCurrentProcedureStep(Wander)

	b.Units[u.id] = u

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

func (b *Blob) GetConsumerNodeID(resourceType ResourceType) (int, error) {
	priorities, ok := b.consumers[resourceType]
	if !ok {
		return 0, errors.New("no consumer found")
	}

	for priority := 0; priority < 10; priority++ {
		ids, ok := priorities[priority]
		if !ok {
			continue
		}

		if len(ids) == 0 {
			continue
		}

		var (
			max   int
			maxID int
			found bool
		)

		for _, id := range ids {
			c := b.Nodes[id].AvailableCapacity()
			if c > max {
				max = c
				maxID = id
				found = true
			}
		}

		if found {
			return maxID, nil
		}
	}

	return 0, errors.New("no consumer found")
}

// GetProducerNodeID returns node id of a producer with highest priority and
// most resources.
func (b *Blob) GetProducerNodeID() (int, ResourceType, error) {
	var (
		max             int
		maxID           int
		maxResourceType ResourceType
		found           bool
	)

	for res := range b.producers {
		nodeID, c, err := b.GetResourceProducerNodeID(res)
		if err != nil {
			continue
		}

		if c > max {
			max = c
			maxID = nodeID
			maxResourceType = res
			found = true
		}
	}

	if !found {
		return 0, "", errors.New("no producer found")
	}

	return maxID, maxResourceType, nil
}

// GetResourceProducerNodeID returns node id of highest priority producer of
// specified resource type with highest amount of resources. resource amount is returned as second return value.
func (b *Blob) GetResourceProducerNodeID(resourceType ResourceType) (int, int, error) {
	priorities, ok := b.producers[resourceType]
	if !ok {
		return 0, 0, errors.New("no producer found")
	}

	for priority := 0; priority < 10; priority++ {
		ids, ok := priorities[priority]
		if !ok {
			continue
		}

		if len(ids) == 0 {
			continue
		}

		var (
			max   int
			maxID int
			found bool
		)

		for _, id := range ids {
			c := b.Nodes[id].ResourceCount(resourceType)
			if c > max {
				max = c
				maxID = id
				found = true
			}
		}

		if found {
			return maxID, max, nil
		}
	}

	return 0, 0, errors.New("no producer found")
}

func (b *Blob) RemoveResources() {
	for _, node := range b.Nodes {
		node.RemoveResources()
	}
}

func (b *Blob) RemoveUnits() {
	b.Units = make(map[int]*Unit)
	b.jobs.Reset()
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
		// TODO: introduce halted job randomly
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

func (jq *JobQueue) Reset() {
	for _, job := range jq.occupied {
		jq.available[job.id] = job
		delete(jq.occupied, job.id)
	}
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
