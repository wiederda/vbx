package main

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

func ReadCSR(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"CSR konnte nicht gelesen werden: %w",
			err,
		)
	}

	block, _ := pem.Decode(data)

	if block == nil {
		return nil, errors.New(
			"CSR enthält keinen gültigen PEM-Block",
		)
	}

	if block.Type != "CERTIFICATE REQUEST" &&
		block.Type != "NEW CERTIFICATE REQUEST" {
		return nil, fmt.Errorf(
			"ungültiger CSR PEM-Typ: %s",
			block.Type,
		)
	}

	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf(
			"CSR konnte nicht geparst werden: %w",
			err,
		)
	}

	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf(
			"CSR-Signatur ist ungültig: %w",
			err,
		)
	}

	if len(csr.DNSNames) == 0 {
		return nil, errors.New(
			"CSR enthält keine DNS-Namen",
		)
	}

	return block.Bytes, nil
}
