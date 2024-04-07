package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	cRand "crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/cbergoon/merkletree"
	"github.com/ethereum/go-ethereum/crypto"
)

const TransactionsPerBlock = 200

type Block struct {
	Index        int
	Timestamp    string
	Transactions []merkletree.Content
	PrevHash     string
	Hash         string
	MerkleRoot   []byte
	Proof        bool
}

type Blockchain struct {
	Chain []Block
}
type Transaction struct {
	Sender   string
	Receiver string
	Amount   float64
}

func (t Transaction) CalculateHash() ([]byte, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(fmt.Sprintf("%s%s%f", t.Sender, t.Receiver, t.Amount))); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

func (t Transaction) Equals(other merkletree.Content) (bool, error) {
	otherT, ok := other.(Transaction)
	if !ok {
		return false, errors.New("not the same Transaction type")
	}
	return t.Sender == otherT.Sender && t.Receiver == otherT.Receiver && t.Amount == otherT.Amount, nil
}

func (bc *Blockchain) AddBlock(newBlock Block) {
	if len(bc.Chain) > 0 {
		newBlock.PrevHash = bc.Chain[len(bc.Chain)-1].Hash
	} else {
		newBlock.PrevHash = ""
	}

	// Creating the Merkle Tree for the transactions
	t, err := merkletree.NewTree(newBlock.Transactions)
	if err != nil {
		log.Fatal(err)
	}
	newBlock.MerkleRoot = t.MerkleRoot()

	// Calculating hash of the block
	newBlock.Hash = calculateHashForBlock(newBlock)

	bc.Chain = append(bc.Chain, newBlock)
}

func (bc *Blockchain) GetBlockByIndex(index int) (Block, error) {
	if index < 0 || index >= len(bc.Chain) {
		return Block{}, fmt.Errorf("block index out of range")
	}
	tempBlock := bc.Chain[index]
	// Verify the entire tree (hashes for each node) is valid
	t, err := merkletree.NewTree(tempBlock.Transactions)
	if err != nil {
		log.Fatal(err)
	}
	vt, err := t.VerifyTree()
	if err != nil {
		log.Fatal(err)
	}
	tempBlock.Proof = vt

	return tempBlock, nil
}

func calculateHashForBlock(block Block) string {
	record := string(rune(block.Index)) + block.Timestamp + block.PrevHash + string(block.MerkleRoot)
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}
func (block *Block) GenerateMerkleProofByTransactionIndex(transactionIndex int) ([]byte, error) {
	if transactionIndex < 0 || transactionIndex >= len(block.Transactions) {
		return nil, fmt.Errorf("transaction index out of range")
	}

	tree, err := merkletree.NewTree(block.Transactions)
	if err != nil {
		return nil, err
	}

	ok, err := tree.VerifyContent(block.Transactions[transactionIndex])
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, fmt.Errorf("failed to verify content, hence cannot generate proof")
	}

	return tree.MerkleRoot(), nil
}

func CreateBlock(index int, transactions []merkletree.Content) Block {
	t, err := merkletree.NewTree(transactions)
	if err != nil {
		log.Fatal(err)
	}
	vt, err := t.VerifyTree()
	if err != nil {
		log.Fatal(err)
	}
	Proof := vt

	return Block{
		Index:        index,
		Timestamp:    time.Now().String(),
		Transactions: transactions,
		Proof:        Proof,
	}
}

func GenerateTransactionsForBlock() []merkletree.Content {
	var transactions []merkletree.Content
	SenderAddress, _ := GenerateEthereumAddress()
	ReceiverAddress, _ := GenerateEthereumAddress()
	for i := 0; i < TransactionsPerBlock; i++ {
		transactions = append(transactions, Transaction{
			Sender:   SenderAddress,
			Receiver: ReceiverAddress,
			Amount:   float64(rand.Intn(100)), // Random amount for demonstration
		})
	}
	return transactions
}

func GenerateEthereumAddress() (string, error) {
	privateKey, err := ecdsa.GenerateKey(crypto.S256(), cRand.Reader)
	if err != nil {
		return "", err
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", errors.New("error casting public key to ECDSA")
	}

	publicKeyBytes := append(publicKeyECDSA.X.Bytes(), publicKeyECDSA.Y.Bytes()...)

	if len(publicKeyBytes) != 64 {
		publicKeyBytes = append(make([]byte, 64-len(publicKeyBytes)), publicKeyBytes...)
	}
	hash := crypto.Keccak256(publicKeyBytes)
	address := hash[len(hash)-20:]

	return "0x" + hex.EncodeToString(address), nil
}

func CalculateBlockSize(block Block) int {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(block)
	if err != nil {
		log.Fatalf("Failed to encode block: %v", err)
	}
	return buf.Len()
}

func (bc *Blockchain) CalculateBlockchainSize() int {
	totalSize := 0
	for _, block := range bc.Chain {
		totalSize += CalculateBlockSize(block)
	}
	return totalSize
}

/*
func main() {
	// Initialize blockchain
	bc := Blockchain{}

	// Simulate transaction data
	transactions := GenerateTransactionsForBlock()

	// Add blocks to the blockchain
	for i := 0; i < 100; i++ {
		block := CreateBlock(i, transactions)
		bc.AddBlock(block)
	}

	// Print the blockchain data
	for _, block := range bc.Chain {
		log.Printf("Block %d, Hash: %s, PrevHash: %s, MerkleRoot: %s", block.Index, block.Hash, block.PrevHash, hex.EncodeToString(block.MerkleRoot))

		// // Verify the entire tree (hashes for each node) is valid
		// t, err := merkletree.NewTree(block.Transactions)
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// vt, err := t.VerifyTree()
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// log.Println("Verify Tree: ", vt)

		// // Verify a specific content in in the tree
		// for _, tx := range block.Transactions {
		// 	vc, err := t.VerifyContent(tx)
		// 	if err != nil {
		// 		log.Fatal(err)
		// 	}
		// 	log.Println("Verify Content: ", vc)
		// }
	}

	tempBlock := bc.Chain[50]
	// Verify the entire tree (hashes for each node) is valid
	t, err := merkletree.NewTree(tempBlock.Transactions)
	if err != nil {
		log.Fatal(err)
	}
	vt, err := t.VerifyTree()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Verify Tree: ", vt)

	// checking out specific transaction within the block

	// Verify a specific content in in the tree
	tempTxn := tempBlock.Transactions[123]
	vc, err := t.VerifyContent(tempTxn)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Verify Content: ", vc)

}
*/
