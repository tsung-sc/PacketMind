package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/packetmind/packetmind/internal/config"
)

func (p *Proxy) loadOrGenerateCA() error {
	settings := p.appSettingsSnapshot()
	if settings == nil {
		settings = config.DefaultPacketMindSettings()
	}
	return p.loadOrGenerateCAFromSettings(settings)
}

func (p *Proxy) loadOrGenerateCAFromSettings(settings *config.AppSettings) error {
	certFile := settings.Cert.CACertFile
	keyFile := settings.Cert.CAKeyFile

	certDir := filepath.Dir(certFile)
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return err
	}

	if _, err := os.Stat(certFile); err == nil {
		return p.loadCA(certFile, keyFile)
	}

	return p.generateCA(certFile, keyFile, settings.Cert.Organization)
}

func (p *Proxy) loadCA(certFile, keyFile string) error {
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return err
	}

	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return err
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to decode certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return fmt.Errorf("failed to decode key")
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return err
	}

	p.caCert = cert
	p.caKey = key
	return nil
}

func (p *Proxy) generateCA(certFile, keyFile, organization string) error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{organization},
			CommonName:   organization + " Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return err
	}

	certOut, err := os.Create(certFile)
	if err != nil {
		return err
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	certOut.Close()

	keyOut, err := os.Create(keyFile)
	if err != nil {
		return err
	}
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	keyOut.Close()

	p.caCert = template
	p.caKey = key

	fmt.Printf("[Proxy] Generated CA certificate: %s\n", certFile)
	return nil
}

func (p *Proxy) getCert(hostname string) (*tls.Certificate, error) {
	if cert, ok := p.certCache.Load(hostname); ok {
		return cert.(*tls.Certificate), nil
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{p.appSettingsSnapshot().Cert.Organization},
			CommonName:   hostname,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(1, 0, 0),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{hostname},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, p.caCert, &key.PublicKey, p.caKey)
	if err != nil {
		return nil, err
	}

	cert := &tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}

	p.certCache.Store(hostname, cert)
	return cert, nil
}

func (p *Proxy) GetCACertPath() string {
	settings := p.appSettingsSnapshot()
	if settings == nil {
		return config.DefaultPacketMindSettings().Cert.CACertFile
	}
	return settings.Cert.CACertFile
}
