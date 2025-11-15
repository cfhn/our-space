package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"os"
)

func main() {
	path := flag.String("path", "./signing_key.pem", "Path to the file to write the private key to")
	publicPath := flag.String("public-path", "./verification_key.pem", "Path to the file to write the public key to")
	flag.Parse()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		panic(err)
	}

	outFile, err := os.Create(*path)
	if err != nil {
		panic(err)
	}

	err = pem.Encode(outFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes})
	if err != nil {
		panic(err)
	}

	err = outFile.Close()
	if err != nil {
		panic(err)
	}

	outFile, err = os.Create(*publicPath)
	if err != nil {
		panic(err)
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		panic(err)
	}

	err = pem.Encode(outFile, &pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes})
	if err != nil {
		panic(err)
	}
}
