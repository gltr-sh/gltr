package cmd

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/mikesmitty/edkey"

	"golang.org/x/crypto/ssh"
)

// generatePrivateKey creates a RSA Private Key of specified byte size
func generateKeyPair() (*ed25519.PublicKey, *ed25519.PrivateKey, error) {
	// Private Key generation
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	return &publicKey, &privateKey, nil
}

// // generatePublicKey take a rsa.PublicKey and return bytes suitable for writing to .pub file
// // returns in the format "ssh-rsa ..."
// func generatePublicKey(privatekey *ed25519.PublicKey) ([]byte, error) {
// 	publicRsaKey, err := ssh.NewPublicKey(privatekey)
// 	if err != nil {
// 		return nil, err
// 	}

// 	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

// 	log.Println("Public key generated")
// 	return pubKeyBytes, nil
// }

// encodePrivateKeyToPEM encodes Private Key from RSA to PEM format
func encodePrivateKeyToPEM(privateKey *ed25519.PrivateKey) []byte {
	// Get ASN.1 DER format
	bytes, err := x509.MarshalPKCS8PrivateKey(*privateKey)
	if err != nil {
		fmt.Printf("Error marshaling private key: %s\n", err.Error())
	}

	// pem.Block
	privBlock := pem.Block{
		// not 100% sure that this is correct...
		Type:    "PRIVATE KEY",
		Headers: nil,
		Bytes:   bytes,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privBlock)

	return privatePEM
}

// func generateKeyPair() (*ed25519.PublicKey, *ed25519.PrivateKey, error) {
// 	// bitSize := defaultKeySize
// 	publicKey, privateKey, err := generateKeys()
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	return publicKey, privateKey, nil
// }

// writePemToFile writes keys to a file
func writeKeysToDirectory(
	keyDirectory, projectID string,
	publicKey *ed25519.PublicKey,
	privateKey *ed25519.PrivateKey,
) error {

	sshPublicKey, _ := ssh.NewPublicKey(*publicKey)

	pemKey := &pem.Block{
		Type:  "OPENSSH PRIVATE KEY",
		Bytes: edkey.MarshalED25519PrivateKey(*privateKey), // <- marshals ed25519 correctly
	}
	pemEncodedPrivateKey := pem.EncodeToMemory(pemKey)
	authorizedKey := ssh.MarshalAuthorizedKey(sshPublicKey)

	err := os.MkdirAll(keyDirectory, 0700)
	if err != nil {
		fmt.Printf("Error creating directory for keys: %v\n", err)
		os.Exit(1)
	}

	privateKeyFilename := projectID
	publicKeyFilename := projectID + ".pub"

	privateKeyLongFilename := path.Join(keyDirectory, privateKeyFilename)

	// _ = ioutil.WriteFile("id_ed25519", pemEncodedPrivateKey, 0600)
	// _ = ioutil.WriteFile("id_ed25519.pub", authorizedKey, 0644)

	err = ioutil.WriteFile(privateKeyLongFilename, pemEncodedPrivateKey, 0600)
	if err != nil {
		fmt.Printf("Error writing private key to file: %v\n", err)
		return err
	}

	publicKeyLongFilename := path.Join(keyDirectory, publicKeyFilename)
	err = ioutil.WriteFile(publicKeyLongFilename, authorizedKey, 0644)
	if err != nil {
		fmt.Printf("Error writing public key to file: %v\n", err)
		return err
	}

	fmt.Printf("Keypair generated for project ID %v\n", projectID)
	return nil
}

// writePemToFile writes keys to a file
func readPrivateKey(gltrConfigDir, projectID string) (privateKey []byte, err error) {
	// private key is stored in config-dir/projectid/projectid
	filename := path.Join(gltrConfigDir, "secrets", projectID, projectID)
	keyBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// log.Printf("Public Key saved to: %s", saveFileTo)
	return keyBytes, nil
}

// writePemToFile writes keys to a file
func writePublicKeyToFile(publicKey []byte, saveFileTo string) error {
	err := ioutil.WriteFile(saveFileTo, publicKey, 0600)
	if err != nil {
		return err
	}

	log.Printf("Public Key saved to: %s", saveFileTo)
	return nil
}
