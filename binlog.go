package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	_ "github.com/go-sql-driver/mysql"
)

func listen() {
	dsn := "root:@tcp(127.0.0.1:3306)/"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var pos mysql.Position
	rows2, err := db.Query("SHOW BINARY LOG STATUS")

	if err != nil {
		log.Fatal(err)
	}
	defer rows2.Close()

	if rows2.Next() {
		var file string
		var position uint32
		rows2.Scan(&file, &position /* ignore rest */)
		pos = mysql.Position{Name: file, Pos: position}
	}

	cfg := replication.BinlogSyncerConfig{
		ServerID: 1,
		Flavor:   "mysql",
		Host:     "127.0.0.1",
		Port:     3306,
		User:     "root",
	}

	syncer := replication.NewBinlogSyncer(cfg)

	streamer, err := syncer.StartSync(pos)

	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	for {
		event, err := streamer.GetEvent(ctx)

		switch ev := event.Event.(type) {

		case *replication.RowsEvent:
			schema := string(ev.Table.Schema)
			table := string(ev.Table.Table)

			if schema != "sakila" {
				continue
			}

			switch event.Header.EventType {

			// INSERT
			case replication.WRITE_ROWS_EVENTv2:
				for _, row := range ev.Rows {
					fmt.Printf("[INSERT] %s.%s → %v\n", schema, table, row)
				}

			// DELETE
			case replication.DELETE_ROWS_EVENTv2:
				for _, row := range ev.Rows {
					fmt.Printf("[DELETE] %s.%s → %v\n", schema, table, row)
				}

			// UPDATE
			case replication.UPDATE_ROWS_EVENTv2:
				for i := 0; i < len(ev.Rows); i += 2 {
					before := ev.Rows[i]
					after := ev.Rows[i+1]

					fmt.Printf("[UPDATE] %s.%s\n", schema, table)
					fmt.Printf("  BEFORE → %v\n", before)
					fmt.Printf("  AFTER  → %v\n", after)
				}
			}
		}

		if err != nil {
			log.Fatal(err)
		}

		var buf bytes.Buffer

		event.Dump(&buf)

		fmt.Printf("%+v | %+v\n", event.Header.EventType, buf.String())
	}
}
