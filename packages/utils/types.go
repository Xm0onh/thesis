package utils

type LTBlock struct {
	BlockCode int64  `json:"blockCode"`
	Data      []byte `json:"data"`
}

type SizeOfMessage struct {
	MessageSize int `json:"messageSize"`
}

type RequestedBlocks struct {
	BlockNumber []int `json:"blockNumber"`
}

type RequestedDroplets struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type SetupParameters struct {
	DegreeCDF       []float64 `json:"degreeCDF"`
	RandomSeed      int64     `json:"randomSeed"`
	SourceBlocks    int       `json:"sourceBlocks"`
	EncodedBlockIDs int       `json:"encodedBlockIDs"`
	NumberOfBlocks  int       `json:"numberOfBlocks"`
	MessageSize     int       `json:"messageSize"`
}

type StartSignal struct {
	Start           bool `json:"start"`
	SourceBlocks    int  `json:"sourceBlocks"`
	EncodedBlockIDs int  `json:"encodedBlockIDs"`
	NumberOfBlocks  int  `json:"numberOfBlocks"`
}
