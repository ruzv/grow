package blob

type ConnectionIDs struct {
	Node1 int `json:"node_1"`
	Node2 int `json:"node_2"`
}

func NewConnectionIDs(id1, id2 int) ConnectionIDs {
	if id1 > id2 {
		return ConnectionIDs{id2, id1}
	}

	return ConnectionIDs{id1, id2}
}

func (c *ConnectionIDs) Opposite(nodeID int) int {
	if c.Node1 == nodeID {
		return c.Node2
	}

	return c.Node1
}

type Connection struct {
	Nodes  ConnectionIDs `json:"nodes"`
	Length float64       `json:"length"`
}
