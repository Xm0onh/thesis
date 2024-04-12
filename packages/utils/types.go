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

type SetupParameters struct {
	DegreeCDF       []float64 `json:"degreeCDF"`
	RandomSeed      int64     `json:"randomSeed"`
	SourceBlocks    int       `json:"sourceBlocks"`
	EncodedBlockIDs int       `json:"encodedBlockIDs"`
	NumberOfBlocks  int       `json:"numberOfBlocks"`
}
