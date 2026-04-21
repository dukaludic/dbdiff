package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"log/slog"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	_ "github.com/go-sql-driver/mysql"
)

type RowEvent struct {
	table     string
	eventType EventType
	data      []any
}

type EventType int

const (
	Insert EventType = iota
	Update
	Delete
	Unknown
)

func mapEventType(e replication.EventType) EventType {
	switch e {
	case replication.WRITE_ROWS_EVENTv2:
		return Insert
	case replication.UPDATE_ROWS_EVENTv2:
		return Update
	case replication.DELETE_ROWS_EVENTv2:
		return Delete
	default:
		return Unknown
	}
}

var tableColumnMap = make(map[string][]string)

func listen(ctx context.Context, out chan<- RowEvent) {
	dsn := "root:@tcp(127.0.0.1:3306)/sakila"
	db, err := sql.Open("mysql", dsn)

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	var pos mysql.Position
	binlogFileAndPosition, err := db.Query("SHOW BINARY LOG STATUS")

	if err != nil {
		log.Fatal(err)
	}
	defer binlogFileAndPosition.Close()

	if binlogFileAndPosition.Next() {
		var file string
		var position uint32

		var ignore1, ignore2, ignore3 any
		if err := binlogFileAndPosition.Scan(&file, &position, &ignore1, &ignore2, &ignore3); err != nil {
			log.Fatal("failed to scan binlog status:", err)
		}
		pos = mysql.Position{Name: file, Pos: position}
	}

	cfg := replication.BinlogSyncerConfig{
		ServerID: 1,
		Flavor:   "mysql",
		Host:     "127.0.0.1",
		Port:     3306,
		User:     "root",
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	syncer := replication.NewBinlogSyncer(cfg)

	streamer, err := syncer.StartSync(pos)

	if err != nil {
		log.Fatal(err)
	}

	for {
		event, err := streamer.GetEvent(ctx)

		if err != nil {
			log.Fatal(err)
		}

		switch ev := event.Event.(type) {

		case *replication.TableMapEvent:
			table := string(ev.Table)

			// fmt.Println("%+v", string(ev.Schema))
			// os.Exit(1)

			// Check if there is better way to do this. Probably not
			schema := string(ev.Schema)
			if schema != "sakila" {
				continue
			}

			// TODO: Optimize. Maybe some caching. See when TableMapEvent is actually required.
			query := `
			SELECT COLUMN_NAME
			FROM information_schema.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE()
			AND TABLE_NAME = ?
			ORDER BY ORDINAL_POSITION
			`

			rows, err := db.Query(query, table)
			if err != nil {
				log.Fatal(err)
			}

			var columns []string
			for rows.Next() {
				var column string
				if err := rows.Scan(&column); err != nil {
					log.Fatal(err)
				}
				columns = append(columns, column)
			}

			// Cache getColumns query here. See when to invalidate
			tableColumnMap[table] = columns

		case *replication.RowsEvent:
			schema := string(ev.Table.Schema)
			table := string(ev.Table.Table)
			if schema != "sakila" {
				continue
			}

			switch mapEventType(event.Header.EventType) {

			// INSERT
			case Insert:
				// for _, row := range ev.Rows {

				// 	fmt.Printf("[INSERT] %s.%s → %v\n", schema, table, row)
				// }

			// DELETE
			case Delete:
				// for _, row := range ev.Rows {
				// 	fmt.Printf("[DELETE] %s.%s → %v\n", schema, table, row)

				// 	out <- RowEvent{
				// 		table:     table,
				// 		eventType: replication.DELETE_ROWS_EVENTv2.String(),
				// 		data:      row,
				// 	}
				// }

			// UPDATE
			case Update:
				for i := 0; i < len(ev.Rows); i += 2 {

					before := ev.Rows[i]
					after := ev.Rows[i+1]

					beforeMapped := make(map[string]interface{})
					afterMapped := make(map[string]interface{})

					for j := 0; j < len(tableColumnMap[table]); j++ {
						beforeMapped[tableColumnMap[table][j]] = toString(before[j])
						afterMapped[tableColumnMap[table][j]] = toString(after[j])
					}

					out <- RowEvent{
						table:     table,
						eventType: Update,
						data: []any{
							beforeMapped,
							afterMapped,
						},
					}
				}
			}

			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func toString(v any) string {
	switch val := v.(type) {
	case nil:
		return "NULL"
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
