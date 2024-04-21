package kzg

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"time"

	kzg "github.com/arnaucube/kzg-commitments-study"
	gthCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/bn256"
	poly "github.com/georgercarder/polynomial"
	lubyTransform "github.com/xm0onh/thesis/packages/luby"
	utils "github.com/xm0onh/thesis/packages/utils"
)

func CalculateKZGParam(ctx context.Context, bucket string, droplets []lubyTransform.LTBlock) {
	var allDropletsData []byte
	for _, droplet := range droplets {
		allDropletsData = append(allDropletsData, droplet.Data...)
	}
	seedHash := gthCrypto.Keccak256(allDropletsData)

	// Seed is deprecated: As of Go 1.20 there is no reason to call Seed with a random value. Programs that call Seed with a known value to get a specific sequence of results should use New(NewSource(seed)) to obtain a local random generator.deprecateddefault
	rand := rand.New(rand.NewSource(Slice2Int64(seedHash)))
	rand.Seed(Slice2Int64(seedHash))
	// Seed the PRNG

	// Convert each droplet into a big.Int based on its data (simplified approach)
	var roots []*big.Int
	for _, droplet := range droplets {
		// Here, we use the hash of the droplet data as a root. Adjust according to your actual data structure.
		hash := gthCrypto.Keccak256(droplet.Data)
		bHash := new(big.Int).SetBytes(hash)
		roots = append(roots, bHash)
	}

	serializedRoots, err := SerializeRoots(roots)
	if err != nil {
		log.Fatalf("Failed to serialize roots: %v", err)
	}

	// Uplaod roots to S3
	utils.UploadToS3(ctx, bucket, "kzg-roots.dat", serializedRoots)

	p := poly.NewPolynomialWithRootsFromArray(roots)

	// Generate TrustedSetup
	ts, err := kzg.NewTrustedSetup(len(p.Coefficients))
	if err != nil {
		log.Fatalf("Failed to generate TrustedSetup: %v", err)
	}

	serializedTS, err := SerializeTrustedSetup(ts)
	if err != nil {
		log.Fatalf("Failed to serialize TrustedSetup: %v", err)
	}

	// Upload TrustedSetup to S3
	utils.UploadToS3(ctx, bucket, "trusted_setup.dat", serializedTS)

	// Generate commitments
	c := kzg.Commit(ts, p.Coefficients)
	serializedCommitments, err := SerializeG1Point(c)
	if err != nil {
		log.Fatalf("Failed to serialize commitments: %v", err)
	}

	// Upload commitments to S3
	utils.UploadToS3(ctx, bucket, "commitments.dat", serializedCommitments)

	var proofs []*bn256.G1
	for _, root := range roots {
		z := root
		y := big.NewInt(0)

		proof, err := kzg.EvaluationProof(ts, p.Coefficients, z, y)
		if err != nil {
			log.Fatalf("Failed to generate evaluation proof: %v", err)
		}
		proofs = append(proofs, proof)
	}

	serializedProofs, err := SerializeG1Points(proofs)
	if err != nil {
		log.Fatalf("Failed to serialize proofs: %v", err)
	}

	// Upload proofs to S3
	utils.UploadToS3(ctx, bucket, "proofs.dat", serializedProofs)

}

func Verification(ctx context.Context, bucketName string) {

	startTime := time.Now()
	// Download roots from S3
	roots, err := utils.DownloadFromS3(ctx, bucketName, "kzg-roots.dat")
	if err != nil {
		log.Fatalf("Failed to download roots: %v", err)
	}

	// Download TrustedSetup from S3
	trustedSetup, err := utils.DownloadFromS3(ctx, bucketName, "trusted_setup.dat")
	if err != nil {
		log.Fatalf("Failed to download TrustedSetup: %v", err)
	}

	// Download commitments from S3
	commitments, err := utils.DownloadFromS3(ctx, bucketName, "commitments.dat")
	if err != nil {
		log.Fatalf("Failed to download commitments: %v", err)
	}

	// Download proofs from S3
	proofs, err := utils.DownloadFromS3(ctx, bucketName, "proofs.dat")
	if err != nil {
		log.Fatalf("Failed to download proofs: %v", err)
	}

	// Deserialize roots
	rootsInts, err := DeserializeRoots(roots)
	if err != nil {
		log.Fatalf("Failed to deserialize roots: %v", err)
	}

	// Deserialize TrustedSetup
	ts, err := DeserializeTrustedSetup(trustedSetup)
	if err != nil {
		log.Fatalf("Failed to deserialize TrustedSetup: %v", err)
	}

	// Deserialize commitments
	commitmentsG1, err := DeserializeG1Point(commitments)
	if err != nil {
		log.Fatalf("Failed to deserialize commitments: %v", err)
	}

	// Deserialize proofs
	proofsG1, err := DeserializeG1Points(proofs)
	if err != nil {
		log.Fatalf("Failed to deserialize proofs: %v", err)
	}
	fmt.Println("Time to download and deserialize: ", time.Since(startTime))
	// Verify proofs
	for i, proof := range proofsG1 {
		z := rootsInts[i]
		y := big.NewInt(0)
		if !kzg.Verify(ts, commitmentsG1, proof, z, y) {
			log.Fatalf("Failed to verify proof")
		}
	}
}
