package kzg

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"math/big"

	kzg "github.com/arnaucube/kzg-commitments-study"
	bn256 "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
)

func Slice2Int64(slc []byte) (r int64) {
	b := new(big.Int)
	b.SetBytes(slc)
	r = b.Int64()
	return
}

func SerializePolynomial(ints []*big.Int) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	// Encode each *big.Int
	for _, n := range ints {
		// Use a string representation to include the sign
		str := n.String() // Convert *big.Int to string
		if err := encoder.Encode(str); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func SerializeRoots(roots []*big.Int) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	for _, root := range roots {
		if err := encoder.Encode(root.Bytes()); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func SerializeTrustedSetup(ts *kzg.TrustedSetup) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	// Marshal Tau1 (G1 points)
	var tau1Bytes [][]byte
	for _, point := range ts.Tau1 {
		tau1Bytes = append(tau1Bytes, point.Marshal())
	}

	// Marshal Tau2 (G2 points)
	var tau2Bytes [][]byte
	for _, point := range ts.Tau2 {
		tau2Bytes = append(tau2Bytes, point.Marshal())
	}

	// Encode the slices of byte slices
	if err := encoder.Encode(tau1Bytes); err != nil {
		return nil, err
	}
	if err := encoder.Encode(tau2Bytes); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// deserializeRoots deserializes a byte slice back into a slice of *big.Int (roots).
func DeserializeRoots(data []byte) ([]*big.Int, error) {
	buf := bytes.NewReader(data)
	decoder := gob.NewDecoder(buf)

	var roots []*big.Int
	for {
		var bytesRoot []byte
		err := decoder.Decode(&bytesRoot)
		if err != nil {
			if err == io.EOF {
				break // End of data
			}
			return nil, err // Handle other errors
		}

		root := new(big.Int).SetBytes(bytesRoot)
		roots = append(roots, root)
	}

	return roots, nil
}

func DeserializePolynomial(data []byte) ([]*big.Int, error) {
	buf := bytes.NewReader(data)
	decoder := gob.NewDecoder(buf)

	var ints []*big.Int
	for {
		var str string
		err := decoder.Decode(&str)
		if err != nil {
			if err == io.EOF {
				break // Reached end of data, exit loop
			}
			return nil, err // An error occurred, return it
		}

		// Convert string back to *big.Int
		n, ok := new(big.Int).SetString(str, 10)
		if !ok {
			return nil, err // Handle conversion error
		}
		ints = append(ints, n)
	}

	return ints, nil
}

// Deserialize a TrustedSetup from a byte slice
func DeserializeTrustedSetup(data []byte) (*kzg.TrustedSetup, error) {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)

	var tau1Bytes, tau2Bytes [][]byte
	if err := decoder.Decode(&tau1Bytes); err != nil {
		return nil, err
	}
	if err := decoder.Decode(&tau2Bytes); err != nil {
		return nil, err
	}

	ts := &kzg.TrustedSetup{
		Tau1: make([]*bn256.G1, len(tau1Bytes)),
		Tau2: make([]*bn256.G2, len(tau2Bytes)),
	}

	// Unmarshal Tau1 (G1 points)
	for i, bytes := range tau1Bytes {
		point := new(bn256.G1)
		if _, err := point.Unmarshal(bytes); err != nil {
			return nil, err
		}
		ts.Tau1[i] = point
	}

	// Unmarshal Tau2 (G2 points)
	for i, bytes := range tau2Bytes {
		point := new(bn256.G2)
		if _, err := point.Unmarshal(bytes); err != nil {
			return nil, err
		}
		ts.Tau2[i] = point
	}

	return ts, nil
}

func SerializeG1Point(point *bn256.G1) ([]byte, error) {
	if point == nil {
		return nil, fmt.Errorf("nil point provided")
	}
	return point.Marshal(), nil
}

func DeserializeG1Point(data []byte) (*bn256.G1, error) {
	point := new(bn256.G1)
	if _, err := point.Unmarshal(data); err != nil {
		return nil, err
	}
	return point, nil
}

func SerializeG1Points(points []*bn256.G1) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	for _, point := range points {
		if err := encoder.Encode(point.Marshal()); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func DeserializeG1Points(data []byte) ([]*bn256.G1, error) {
	buf := bytes.NewReader(data)
	decoder := gob.NewDecoder(buf)

	var points []*bn256.G1
	for {
		var bytesPoint []byte
		err := decoder.Decode(&bytesPoint)
		if err != nil {
			if err == io.EOF {
				break // End of data
			}
			return nil, err // Handle other errors
		}

		point := new(bn256.G1)
		if _, err := point.Unmarshal(bytesPoint); err != nil {
			return nil, err
		}
		points = append(points, point)
	}

	return points, nil
}
