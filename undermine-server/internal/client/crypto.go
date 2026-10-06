package client

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

type KeyPair struct {
	PrivateKey [32]byte
	PublicKey  [32]byte
}

func GenerateKeyPair() (*KeyPair, error) {
	var pair KeyPair
	_, err := rand.Read(pair.PrivateKey[:])
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации случайных байт: %w", err)
	}
	curve25519.ScalarBaseMult(&pair.PublicKey, &pair.PrivateKey)
	return &pair, nil
}

func ComputeSharedKey(myPrivate [32]byte, theirPublic [32]byte) ([32]byte, error) {
	var sharedKey [32]byte
	curve25519.ScalarMult(&sharedKey, &myPrivate, &theirPublic)
	return sharedKey, nil
}

func Encrypt(plaintext string, sharedKey [32]byte) (nonce [12]byte, ciphertext []byte, err error) {
	_, err = rand.Read(nonce[:])
	if err != nil {
		return nonce, nil, err
	}

	aead, err := chacha20poly1305.New(sharedKey[:])
	if err != nil {
		return nonce, nil, err
	}

	ciphertext = aead.Seal(nil, nonce[:], []byte(plaintext), nil)
	return nonce, ciphertext, nil
}

func Decrypt(nonce [12]byte, ciphertext []byte, sharedKey [32]byte) (string, error) {
	aead, err := chacha20poly1305.New(sharedKey[:])
	if err != nil {
		return "", err
	}

	plaintextBytes, err := aead.Open(nil, nonce[:], ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintextBytes), nil
}
