package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	_ "github.com/go-sql-driver/mysql"
)

type RowEvent struct {
	table     string
	eventType string
	data      []interface{}
}

type MappedColumnData struct {
	column string
	data   any
}

func listen(ctx context.Context, out chan<- RowEvent) {
	dsn := "root:@tcp(127.0.0.1:3306)/sakila"
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

	tableColumnMap := make(map[string][]string)

	for {
		event, err := streamer.GetEvent(ctx)

		switch ev := event.Event.(type) {

		case *replication.TableMapEvent:
			table := string(ev.Table)

			schema := string(ev.Schema)
			if schema != "sakila" {
				continue
			}

			query := fmt.Sprintf("SHOW COLUMNS FROM `%s`", table)

			rows, err := db.Query(query)

			if err != nil {
				log.Fatal(err)
			}

			columns := []string{}

			for rows.Next() {
				var field, discard, discard2, key, discard3, discard4 sql.NullString

				err := rows.Scan(&field, &discard, &discard2, &key, &discard3, &discard4)
				if err != nil {
					panic(err)
				}

				var column string
				if field.Valid {
					column = field.String
				} else {
					column = ""
				}

				columns = append(columns, column)
			}

			tableColumnMap[table] = columns

		case *replication.RowsEvent:
			schema := string(ev.Table.Schema)
			table := string(ev.Table.Table)
			if schema != "sakila" {
				continue
			}

			switch event.Header.EventType {

			// INSERT
			case replication.WRITE_ROWS_EVENTv2:
				// for _, row := range ev.Rows {

				// 	fmt.Printf("[INSERT] %s.%s → %v\n", schema, table, row)
				// }

			// DELETE
			case replication.DELETE_ROWS_EVENTv2:
				// for _, row := range ev.Rows {
				// 	fmt.Printf("[DELETE] %s.%s → %v\n", schema, table, row)

				// 	out <- RowEvent{
				// 		table:     table,
				// 		eventType: replication.DELETE_ROWS_EVENTv2.String(),
				// 		data:      row,
				// 	}
				// }

			// UPDATE
			case replication.UPDATE_ROWS_EVENTv2:
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
						eventType: replication.DELETE_ROWS_EVENTv2.String(),
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
