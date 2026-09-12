package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      uint32 = 64 * 1024
	argonIterations  uint32 = 3
	argonParallelism uint8  = 1
	argonSaltBytes          = 16
	argonKeyBytes           = 32
)

type Argon2idHasher struct{}

func NewArgon2idHasher() Argon2idHasher {
	return Argon2idHasher{}
}

func (Argon2idHasher) Hash(password string) (string, error) {
	salt := make([]byte, argonSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyBytes)
	encode := base64.RawStdEncoding.EncodeToString
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", argonMemory, argonIterations, argonParallelism, encode(salt), encode(key)), nil
}

func (Argon2idHasher) Compare(encodedHash, password string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}
	parameters := map[string]uint64{}
	for _, parameter := range strings.Split(parts[3], ",") {
		pieces := strings.SplitN(parameter, "=", 2)
		if len(pieces) != 2 {
			return false
		}
		value, err := strconv.ParseUint(pieces[1], 10, 32)
		if err != nil {
			return false
		}
		parameters[pieces[0]] = value
	}
	memory, memoryOK := parameters["m"]
	iterations, iterationsOK := parameters["t"]
	parallelism, parallelismOK := parameters["p"]
	if !memoryOK || !iterationsOK || !parallelismOK || memory == 0 || iterations == 0 || parallelism == 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, uint32(iterations), uint32(memory), uint8(parallelism), uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}
