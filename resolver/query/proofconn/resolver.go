package proofconn

import (
	"context"

	"github.com/findy-network/findy-agent-vault/db/store"
	"github.com/findy-network/findy-agent-vault/graph/model"
	"github.com/findy-network/findy-agent-vault/resolver/query/agent"
	"github.com/findy-network/findy-agent-vault/utils"
	"github.com/lainio/err2"
	"github.com/lainio/err2/try"
)

type Resolver struct {
	db store.DB
	*agent.Resolver
}

func NewResolver(db store.DB, agentResolver *agent.Resolver) *Resolver {
	return &Resolver{db, agentResolver}
}

func (r *Resolver) TotalCount(ctx context.Context, obj *model.ProofConnection) (c int, err error) {
	defer err2.Return(&err)

	tenant := try.To1(r.GetAgent(ctx))

	utils.LogLow().Infof(
		"proofConnectionResolver:TotalCount for tenant %s, connection: %v",
		tenant.ID,
		obj.ConnectionID,
	)

	count := try.To1(r.db.GetProofCount(tenant.ID, obj.ConnectionID))

	return count, nil
}
