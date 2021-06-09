package test

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/findy-network/findy-agent-vault/db/model"
	"github.com/findy-network/findy-agent-vault/db/store"
	"github.com/findy-network/findy-agent-vault/db/store/pg"
	graph "github.com/findy-network/findy-agent-vault/graph/model"
	"github.com/findy-network/findy-agent-vault/utils"
	"github.com/google/uuid"
)

const testAgentLabel = "testAgent"

type testableDB struct {
	db               store.DB
	name             string
	testTenantID     string
	testAgentID      string
	testConnectionID string
	testConnection   *model.Connection
}

func (t *testableDB) updateTestConnection() {
	c := &model.Connection{}
	*c = *t.testConnection
	c.ID = uuid.New().String()
	t.testConnection = c
}

func (t *testableDB) newTestCredential(cred *model.Credential) *model.Credential {
	c := &model.Credential{}
	*c = *cred
	c.ID = uuid.New().String()
	c.TenantID = t.testTenantID
	c.ConnectionID = t.testConnectionID
	return c
}

func (t *testableDB) newTestEvent(event *model.Event) *model.Event {
	e := &model.Event{}
	*e = *event
	e.ID = uuid.New().String()
	e.TenantID = t.testTenantID
	e.ConnectionID = &t.testConnectionID
	return e
}

func (t *testableDB) newTestJob(job *model.Job) *model.Job {
	j := &model.Job{}
	*j = *job
	j.ID = uuid.New().String()
	j.TenantID = t.testTenantID
	j.ConnectionID = &t.testConnectionID
	return j
}

func (t *testableDB) newTestMessage(msg *model.Message) *model.Message {
	m := &model.Message{}
	*m = *msg
	m.ID = uuid.New().String()
	m.TenantID = t.testTenantID
	m.ConnectionID = t.testConnectionID
	return m
}

func (t *testableDB) newTestProof(proof *model.Proof) *model.Proof {
	p := &model.Proof{}
	*p = *proof
	p.ID = uuid.New().String()
	p.TenantID = t.testTenantID
	p.ConnectionID = t.testConnectionID
	return p
}

var (
	DBs            []*testableDB
	testCredential *model.Credential = &model.Credential{
		Base:          model.Base{ID: uuid.New().String()},
		Role:          graph.CredentialRoleHolder,
		SchemaID:      "schemaId",
		CredDefID:     "credDefId",
		InitiatedByUs: false,
		Attributes: []*graph.CredentialValue{
			{Name: "name1", Value: "value1"},
			{Name: "name2", Value: "value2"},
		},
	}
	testProof *model.Proof = &model.Proof{
		Role:          graph.ProofRoleProver,
		InitiatedByUs: false,
		Result:        true,
		Attributes: []*graph.ProofAttribute{
			{Name: "name1", CredDefID: "cred_def_id"},
			{Name: "name2", CredDefID: "cred_def_id"},
		},
	}
	testMessage *model.Message = &model.Message{Message: "msg content",
		SentByMe:  false,
		Delivered: nil,
	}

	testEvent *model.Event = &model.Event{
		Base:        model.Base{ID: uuid.New().String()},
		Description: "event desc",
		Read:        false,
	}
	testJob *model.Job = &model.Job{
		ProtocolType: graph.ProtocolTypeConnection,
		Status:       graph.JobStatusWaiting,
		Result:       graph.JobResultNone,
	}
)

func validateTimestap(t *testing.T, exp, got *time.Time, name string) {
	fail := false
	if got != exp {
		fail = true
		if got != nil && exp != nil && got.Sub(*exp) < (time.Microsecond*100) {
			fail = false
		}
	}
	if fail {
		t.Errorf("Object %s mismatch expected %s got %s", name, exp, got)
	}
}

func validateStrPtr(t *testing.T, exp, got *string, name string) {
	fail := false
	if got != exp {
		fail = true
		if got != nil && exp != nil {
			if *got != *exp {
				t.Errorf("Object %s mismatch expected %s got %s", name, *exp, *got)
			} else {
				fail = false
			}
		}
	}
	if fail {
		t.Errorf("Object %s mismatch expected %v got %v", name, exp, got)
	}
}

func setup() {
	logLevel := "5"
	logQueries := false
	if logQueries {
		logLevel = "7"
	}
	utils.SetLogConfig(&utils.Configuration{LogLevel: logLevel})

	testAgent := &model.Agent{}
	testAgent.AgentID = "testAgentID"
	testAgent.Label = testAgentLabel

	testConnection := &model.Connection{}
	testConnection.OurDid = "ourDid"
	testConnection.TheirDid = "theirDid"
	testConnection.TheirEndpoint = "theirEndpoint"
	testConnection.TheirLabel = "theirLabel"
	testConnection.Invited = false

	DBs = append(DBs, []*testableDB{{
		db: pg.InitDB(
			&utils.Configuration{
				DBHost:           "localhost",
				DBPassword:       os.Getenv("FAV_DB_PASSWORD"),
				DBPort:           5433,
				DBTracing:        logQueries,
				DBMigrationsPath: "file://../../migrations",
				DBName:           "vault",
			},
			true, false),
		name:           "pg",
		testConnection: testConnection,
	},
	}...)

	for index := range DBs {
		s := DBs[index]

		a, err := s.db.AddAgent(testAgent)
		if err != nil {
			panic(err)
		}
		s.testTenantID = a.ID
		s.testAgentID = a.AgentID

		s.testConnection = testConnection
		s.updateTestConnection()
		s.testConnection.TenantID = s.testTenantID

		c, err := s.db.AddConnection(s.testConnection)
		if err != nil {
			panic(err)
		}
		s.testConnectionID = c.ID
		s.testConnection = c
	}
}

func teardown() {
	for _, s := range DBs {
		s.db.Close()
	}
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func validateCreatedTS(t *testing.T, cursor uint64, ts *time.Time) {
	if time.Since(*ts) > time.Second {
		t.Errorf("Timestamp not in threshold %v", ts)
	}
	created := model.TimeToCursor(ts)
	diff := uint64(math.Abs(float64(cursor) - float64(created)))
	if diff > 1 {
		t.Errorf("Cursor mismatch %v %v (%v)", cursor, created, diff)
	}
}
