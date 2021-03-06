/*
Taken from  https://github.com/kubernetes/kubernetes/blob/master/cmd/kubeadm/app/phases/certs/pkiutil/pki_helpers.go

Added:
 TryLoadPublicKeyFromDisk to support loading the public SA key...
 TryLoadAnyCertFromDisk
 TryLoadAnyKeyFromDisk

 Internalised certutil (from k8s.io/client-go/util/cert)
 */

/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pkiutil

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	certutil "github.com/UKHomeOffice/keto-k8/pkg/client-go/util/cert"
)

// TODO: It should be able to generate different types of private keys, at least: RSA and ECDSA (and in the future maybe Ed25519 as well)
// TODO: See if it makes sense to move this package directly to pkg/util/cert

// NewCertificateAuthority create a new CA
func NewCertificateAuthority() (*x509.Certificate, *rsa.PrivateKey, error) {
	key, err := certutil.NewPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key [%v]", err)
	}

	config := certutil.Config{
		CommonName: "kubernetes",
	}
	cert, err := certutil.NewSelfSignedCACert(config, key)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create self-signed certificate [%v]", err)
	}

	return cert, key, nil
}

// NewCertAndKey - new cert and key
func NewCertAndKey(caCert *x509.Certificate, caKey *rsa.PrivateKey, config certutil.Config) (*x509.Certificate, *rsa.PrivateKey, error) {
	key, err := certutil.NewPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key [%v]", err)
	}

	cert, err := certutil.NewSignedCert(config, key, caCert, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to sign certificate [%v]", err)
	}

	return cert, key, nil
}

// WriteCertAndKey - save new key and cert to disk at pkiPath as name
func WriteCertAndKey(pkiPath string, name string, cert *x509.Certificate, key *rsa.PrivateKey) error {
	if err := WriteKey(pkiPath, name, key); err != nil {
		return err
	}

	if err := WriteCert(pkiPath, name, cert); err != nil {
		return err
	}

	return nil
}

// WriteCert - writes a public cert as name at pkiPath
func WriteCert(pkiPath, name string, cert *x509.Certificate) error {
	if cert == nil {
		return fmt.Errorf("certificate cannot be nil when writing to file")
	}

	certificatePath := pathForCert(pkiPath, name)
	if err := certutil.WriteCert(certificatePath, certutil.EncodeCertPEM(cert)); err != nil {
		return fmt.Errorf("unable to write certificate to file %q: [%v]", certificatePath, err)
	}

	return nil
}

// WriteKey - writes a private key as name at pkiPath
func WriteKey(pkiPath, name string, key *rsa.PrivateKey) error {
	if key == nil {
		return fmt.Errorf("private key cannot be nil when writing to file")
	}

	privateKeyPath := pathForKey(pkiPath, name)
	if err := certutil.WriteKey(privateKeyPath, certutil.EncodePrivateKeyPEM(key)); err != nil {
		return fmt.Errorf("unable to write private key to file %q: [%v]", privateKeyPath, err)
	}

	return nil
}

// WritePublicKey - writes a public key as name at pkiPath
func WritePublicKey(pkiPath, name string, key *rsa.PublicKey) error {
	if key == nil {
		return fmt.Errorf("public key cannot be nil when writing to file")
	}

	publicKeyBytes, err := certutil.EncodePublicKeyPEM(key)
	if err != nil {
		return err
	}
	publicKeyPath := pathForPublicKey(pkiPath, name)
	if err := certutil.WriteKey(publicKeyPath, publicKeyBytes); err != nil {
		return fmt.Errorf("unable to write public key to file %q: [%v]", publicKeyPath, err)
	}

	return nil
}

// CertOrKeyExist retuns a boolean whether the cert or the key exists
func CertOrKeyExist(pkiPath, name string) bool {
	certificatePath, privateKeyPath := pathsForCertAndKey(pkiPath, name)

	_, certErr := os.Stat(certificatePath)
	_, keyErr := os.Stat(privateKeyPath)
	if os.IsNotExist(certErr) && os.IsNotExist(keyErr) {
		// The cert or the key did not exist
		return false
	}

	// Both files exist or one of them
	return true
}

// TryLoadCertAndKeyFromDisk tries to load a cert and a key from the disk and validates that they are valid
func TryLoadCertAndKeyFromDisk(pkiPath, name string) (*x509.Certificate, *rsa.PrivateKey, error) {
	cert, err := TryLoadCertFromDisk(pkiPath, name)
	if err != nil {
		return nil, nil, err
	}

	key, err := TryLoadKeyFromDisk(pkiPath, name)
	if err != nil {
		return nil, nil, err
	}

	return cert, key, nil
}

// TryLoadCertFromDisk tries to load the cert from the disk and validates that it is valid
func TryLoadCertFromDisk(pkiPath, name string) (*x509.Certificate, error) {
	certificatePath := pathForCert(pkiPath, name)
	cert, err := TryLoadAnyCertFromDisk(certificatePath)
	return cert, err
}

// TryLoadAnyCertFromDisk tries to load the cert from the disk and validates that it is valid
func TryLoadAnyCertFromDisk(certificatePath string) (*x509.Certificate, error) {
	certs, err := certutil.CertsFromFile(certificatePath)
	if err != nil {
		return nil, fmt.Errorf("couldn't load the certificate file %s: %v", certificatePath, err)
	}

	// We are only putting one certificate in the certificate pem file, so it's safe to just pick the first one
	// TODO: Support multiple certs here in order to be able to rotate certs
	cert := certs[0]

	// Check so that the certificate is valid now
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return nil, fmt.Errorf("the certificate is not valid yet")
	}
	if now.After(cert.NotAfter) {
		return nil, fmt.Errorf("the certificate has expired")
	}

	return cert, nil
}

// TryLoadKeyFromDisk tries to load the key from the disk and validates that it is valid
func TryLoadKeyFromDisk(pkiPath, name string) (*rsa.PrivateKey, error) {
	privateKeyPath := pathForKey(pkiPath, name)
	key, err := TryLoadAnyKeyFromDisk(privateKeyPath)
	return key, err
}

// TryLoadAnyKeyFromDisk tries to load the key from the disk and validates that it is valid
func TryLoadAnyKeyFromDisk(privateKeyPath string) (*rsa.PrivateKey, error) {

	// Parse the private key from a file
	privKey, err := certutil.PrivateKeyFromFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't load the private key file %s: %v", privateKeyPath, err)
	}

	// Allow RSA format only
	var key *rsa.PrivateKey
	switch k := privKey.(type) {
	case *rsa.PrivateKey:
		key = k
	default:
		return nil, fmt.Errorf("the private key file %s isn't in RSA format", privateKeyPath)
	}

	return key, nil
}

// TryLoadPublicKeyFromDisk - will verify a Public key and return it if OK
func TryLoadPublicKeyFromDisk(pkiPath, name string) (*rsa.PublicKey, error) {
	publicKeyPath := pathForPublicKey(pkiPath, name)

	pubBytes, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pubBytes)
	pubkeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	pubkey, ok := pubkeyInterface.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Error parsing public key %q", publicKeyPath)
	}
	return pubkey, nil
}

func pathsForCertAndKey(pkiPath, name string) (string, string) {
	return pathForCert(pkiPath, name), pathForKey(pkiPath, name)
}

func pathForCert(pkiPath, name string) string {
	return path.Join(pkiPath, fmt.Sprintf("%s.crt", name))
}

func pathForKey(pkiPath, name string) string {
	return path.Join(pkiPath, fmt.Sprintf("%s.key", name))
}

func pathForPublicKey(pkiPath, name string) string {
	return path.Join(pkiPath, fmt.Sprintf("%s.pub", name))
}