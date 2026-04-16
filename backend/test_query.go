package main
import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)
func mainTest() {
	rows, err := db.Query("SELECT ip FROM routes_table WHERE status='published' AND miss_count >= 3 LIMIT 50")
	if err != nil { fmt.Println(err); return }
	count := 0
	for rows.Next() {
		count++
	}
	fmt.Println("count=", count)
}
