package main

import (
	"database/sql"
	"fmt"
	"log"
	"proxygw/remote_deploy"
    _ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "/root/proxygw/config/proxygw.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	var cred, auth string
	db.QueryRow("SELECT ssh_credential, ssh_auth_type FROM remote_nodes WHERE ssh_host='192.168.20.152'").Scan(&cred, &auth)
	
	client, err := remote_deploy.Connect("192.168.20.152", 22, "root", auth, cred)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	
	stdout, stderr, err := client.RunCommand("systemctl status xray --no-pager")
	fmt.Println("STDOUT:", stdout)
	fmt.Println("STDERR:", stderr)
	fmt.Println("ERR:", err)

	stdout, stderr, err = client.RunCommand("journalctl -u xray -n 20 --no-pager")
	fmt.Println("LOGS:", stdout)
}
