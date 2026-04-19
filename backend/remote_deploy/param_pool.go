package remote_deploy

import (
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var (
	poolMutex sync.Mutex
	rng       = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// GenerateUniquePort finds an unused port between min and max that is not used by local nodes or remote nodes
func GenerateUniquePort(db *sql.DB, minPort, maxPort int) (int, error) {
	poolMutex.Lock()
	defer poolMutex.Unlock()

	// Gather used ports
	usedPorts := make(map[int]bool)

	// Check existing remote WG nodes
	rowsWg, err := db.Query("SELECT port FROM remote_node_wg")
	if err == nil {
		defer rowsWg.Close()
		for rowsWg.Next() {
			var p int
			if err := rowsWg.Scan(&p); err == nil {
				usedPorts[p] = true
			}
		}
	}

	// Check existing remote VLESS nodes
	rowsVl, err := db.Query("SELECT port FROM remote_node_vless")
	if err == nil {
		defer rowsVl.Close()
		for rowsVl.Next() {
			var p int
			if err := rowsVl.Scan(&p); err == nil {
				usedPorts[p] = true
			}
		}
	}

	// Try up to 100 times to find a random free port
	for i := 0; i < 100; i++ {
		p := minPort + rng.Intn(maxPort-minPort+1)
		if !usedPorts[p] {
			return p, nil
		}
	}

	return 0, fmt.Errorf("no available port found in range %d-%d", minPort, maxPort)
}

// GenerateUniqueWGTunnel finds a unique 10.x.x.x/24 or /32 subnet for WG
func GenerateUniqueWGTunnel(db *sql.DB) (string, string, error) {
	poolMutex.Lock()
	defer poolMutex.Unlock()

	usedNets := make(map[string]bool)
	rows, err := db.Query("SELECT tunnel_addr FROM remote_node_wg")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var addr string
			if err := rows.Scan(&addr); err == nil {
				// addr is usually 10.x.y.1/24, we just map the string
				usedNets[addr] = true
			}
		}
	}

	for i := 0; i < 200; i++ {
		x := 10 + rng.Intn(200) // 10.10.x to 10.210.x
		y := rng.Intn(254)

		serverAddr := fmt.Sprintf("10.%d.%d.1/24", x, y)
		clientAddr := fmt.Sprintf("10.%d.%d.2/32", x, y)

		if !usedNets[serverAddr] {
			return serverAddr, clientAddr, nil
		}
	}

	return "", "", fmt.Errorf("failed to generate unique wg tunnel subnet")
}
