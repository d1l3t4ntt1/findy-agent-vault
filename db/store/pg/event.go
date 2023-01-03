package pg

import (
	"database/sql"
	"sort"

	"github.com/findy-network/findy-agent-vault/db/model"
	"github.com/findy-network/findy-agent-vault/paginator"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

const (
	sqlEventFields = "tenant_id, connection_id, job_id, description, read"
	sqlEventSelect = "SELECT id, " + sqlEventFields + ", created, cursor FROM"
)

var (
	sqlEventInsert = "INSERT INTO event " + "(" + sqlEventFields + ") " +
		"VALUES ($1, $2, $3, $4, $5) RETURNING " + sqlInsertFields
)

func (pg *Database) AddEvent(e *model.Event) (event *model.Event, err error) {
	defer err2.Handle(&err, "AddEvent")

	event = &model.Event{}
	*event = *e
	try.To(pg.doRowQuery(
		func(rows *sql.Rows) error {
			return rows.Scan(&event.ID, &event.Created, &event.Cursor)
		},
		sqlEventInsert,
		e.TenantID,
		e.ConnectionID,
		e.JobID,
		e.Description,
		e.Read,
	))

	return event, err
}

func (pg *Database) MarkEventRead(id, tenantID string) (event *model.Event, err error) {
	defer err2.Handle(&err, "MarkEventRead")

	const sqlEventUpdate = "UPDATE event SET read=true WHERE id = $1 AND tenant_id = $2" +
		" RETURNING id," + sqlEventFields + ", created, cursor"

	event = &model.Event{}
	try.To(pg.doRowQuery(
		readRowToEvent(event),
		sqlEventUpdate,
		id,
		tenantID,
	))

	return event, err
}

func rowToEvent(rows *sql.Rows) (event *model.Event, err error) {
	event = &model.Event{}
	return event, readRowToEvent(event)(rows)
}

func readRowToEvent(n *model.Event) func(*sql.Rows) error {
	return func(rows *sql.Rows) error {
		return rows.Scan(
			&n.ID,
			&n.TenantID,
			&n.ConnectionID,
			&n.JobID,
			&n.Description,
			&n.Read,
			&n.Created,
			&n.Cursor,
		)
	}
}

func (pg *Database) GetEvent(id, tenantID string) (event *model.Event, err error) {
	defer err2.Handle(&err, "GetEvent")

	const sqlEventSelectByID = sqlEventSelect + " event" +
		" WHERE event.id=$1 AND tenant_id=$2"

	event = &model.Event{}
	try.To(pg.doRowQuery(
		readRowToEvent(event),
		sqlEventSelectByID,
		id,
		tenantID,
	))

	return
}

func (pg *Database) getEventsForQuery(
	queries *queryInfo,
	batch *paginator.BatchInfo,
	tenantID string,
	initialArgs []interface{},
) (e *model.Events, err error) {
	defer err2.Handle(&err, "GetEvents")

	query, args := getBatchQuery(queries, batch, tenantID, initialArgs)
	e = &model.Events{
		Events:          make([]*model.Event, 0),
		HasNextPage:     false,
		HasPreviousPage: false,
	}

	var event *model.Event
	try.To(pg.doRowsQuery(func(rows *sql.Rows) (err error) {
		defer err2.Handle(&err)
		event = try.To1(rowToEvent(rows))
		e.Events = append(e.Events, event)
		return
	}, query, args...))

	if batch.Count < len(e.Events) {
		e.Events = e.Events[:batch.Count]
		if batch.Tail {
			e.HasPreviousPage = true
		} else {
			e.HasNextPage = true
		}
	}

	if batch.After > 0 {
		e.HasPreviousPage = true
	}
	if batch.Before > 0 {
		e.HasNextPage = true
	}

	// Reverse order for tail first
	if batch.Tail {
		sort.Slice(e.Events, func(i, j int) bool {
			return e.Events[i].Created.Sub(e.Events[j].Created) < 0
		})
	}

	return e, err
}

func sqlEventBatchWhere(cursorParam, connectionParam, limitParam string, desc, before bool) string {
	const whereTenantID = " WHERE tenant_id=$1 "
	cursorOrder := sqlOrderByCursorAsc
	cursor := ""
	connection := ""
	compareChar := sqlGreaterThan
	if before {
		compareChar = sqlLessThan
	}
	if connectionParam != "" {
		connection = " AND connection_id = " + connectionParam + " "
	}
	if cursorParam != "" {
		cursor = " AND cursor " + compareChar + cursorParam + " "
		if desc {
			cursor = " AND cursor " + compareChar + cursorParam + " "
		}
	}
	if desc {
		cursorOrder = sqlOrderByCursorDesc
	}
	where := whereTenantID + cursor + connection
	return sqlEventSelect + " event " + where + cursorOrder + " " + limitParam
}

func (pg *Database) GetEvents(info *paginator.BatchInfo, tenantID string, connectionID *string) (c *model.Events, err error) {
	if connectionID == nil {
		return pg.getEventsForQuery(&queryInfo{
			Asc:        sqlEventBatchWhere("", "", "$2", false, false),
			Desc:       sqlEventBatchWhere("", "", "$2", true, false),
			AfterAsc:   sqlEventBatchWhere("$2", "", "$3", false, false),
			AfterDesc:  sqlEventBatchWhere("$2", "", "$3", true, false),
			BeforeAsc:  sqlEventBatchWhere("$2", "", "$3", false, true),
			BeforeDesc: sqlEventBatchWhere("$2", "", "$3", true, true),
		},
			info,
			tenantID,
			[]interface{}{},
		)
	}
	return pg.getEventsForQuery(&queryInfo{
		Asc:        sqlEventBatchWhere("", "$2", "$3", false, false),
		Desc:       sqlEventBatchWhere("", "$2", "$3", true, false),
		AfterAsc:   sqlEventBatchWhere("$2", "$3", "$4", false, false),
		AfterDesc:  sqlEventBatchWhere("$2", "$3", "$4", true, false),
		BeforeAsc:  sqlEventBatchWhere("$2", "$3", "$4", false, true),
		BeforeDesc: sqlEventBatchWhere("$2", "$3", "$4", true, true),
	},
		info,
		tenantID,
		[]interface{}{*connectionID},
	)
}

func (pg *Database) GetEventCount(tenantID string, connectionID *string) (count int, err error) {
	defer err2.Handle(&err, "GetEventCount")
	const (
		sqlEventBatchWhere           = " WHERE tenant_id=$1 "
		sqlEventBatchWhereConnection = " WHERE tenant_id=$1 AND connection_id=$2"
	)
	count = try.To1(pg.getCount(
		"event",
		sqlEventBatchWhere,
		sqlEventBatchWhereConnection,
		tenantID,
		connectionID,
	))
	return
}

func (pg *Database) GetConnectionForEvent(id, tenantID string) (*model.Connection, error) {
	return pg.getConnectionForObject("event", "connection_id", id, tenantID)
}

func (pg *Database) GetJobForEvent(id, tenantID string) (*model.Job, error) {
	return pg.getJobForObject("event", id, tenantID)
}
