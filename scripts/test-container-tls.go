// Simple test to verify TLS works with Docker service names
// This can be run inside a Docker container to test inter-container TLS
package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	certFile := flag.String("cert", "/app/certs/server.crt", "Certificate file")
	target := flag.String("target", "https://api:11000/v1/users", "Target URL to test")
	flag.Parse()

	fmt.Printf("Testing TLS connection to: %s\n", *target)
	fmt.Printf("Using certificate: %s\n\n", *certFile)

	// Load the certificate
	certPEM, err := os.ReadFile(*certFile)
	if err != nil {
		fmt.Printf("❌ Failed to read certificate: %v\n", err)
		os.Exit(1)
	}

	// Create cert pool
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certPEM) {
		fmt.Println("❌ Failed to parse certificate")
		os.Exit(1)
	}

	// Create TLS config with cert pool
	tlsConfig := &tls.Config{
		RootCAs:    certPool,
		MinVersion: tls.VersionTLS12,
	}

	// Create HTTP client
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// Make request
	resp, err := client.Get(*target)
	if err != nil {
		fmt.Printf("❌ Request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Printf("✅ TLS connection successful!\n")
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Response: %s\n", string(body))
}
