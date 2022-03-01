package gqlserver

import (
	"context"
	"sync"

	"github.com/functionalfoundry/graphqlws"
	"github.com/graphql-go/graphql"
)

type subUpdater struct {
	sub    *graphqlws.Subscription
	cancel context.CancelFunc
}

// subManager enhances graphqlws.SubscriptionManager.
type subManager struct {
	ctx    context.Context
	schema *graphql.Schema
	inner  graphqlws.SubscriptionManager

	mutex    sync.Mutex
	updaters map[string]*subUpdater
}

// Subscriptions implements graphqlws.SubscriptionManager interface.
func (m *subManager) Subscriptions() graphqlws.Subscriptions {
	return m.inner.Subscriptions()
}

// AddSubscription implements graphqlws.SubscriptionManager interface.
func (m *subManager) AddSubscription(conn graphqlws.Connection, sub *graphqlws.Subscription) []error {
	errs := m.inner.AddSubscription(conn, sub)
	if len(errs) == 0 {
		m.addSub(conn, sub)
	}
	return errs
}

// RemoveSubscription implements graphqlws.SubscriptionManager interface.
func (m *subManager) RemoveSubscription(conn graphqlws.Connection, sub *graphqlws.Subscription) {
	m.removeSub(conn, sub)
	m.inner.RemoveSubscription(conn, sub)
}

// RemoveSubscriptions implements graphqlws.SubscriptionManager interface.
func (m *subManager) RemoveSubscriptions(conn graphqlws.Connection) {
	for _, sub := range m.Subscriptions()[conn] {
		m.removeSub(conn, sub)
	}
	m.inner.RemoveSubscriptions(conn)
}

func (m *subManager) makeID(conn graphqlws.Connection, sub *graphqlws.Subscription) string {
	return conn.ID() + ":" + sub.ID
}

func (m *subManager) addSub(conn graphqlws.Connection, sub *graphqlws.Subscription) {
	id := m.makeID(conn, sub)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	ctx, cancel := context.WithCancel(m.ctx)
	m.updaters[id] = &subUpdater{
		sub:    sub,
		cancel: cancel,
	}

	results := graphql.ExecuteSubscription(graphql.ExecuteParams{
		Schema:        *m.schema,
		AST:           sub.Document,
		OperationName: sub.OperationName,
		Args:          sub.Variables,
		Context:       ctx,
	})

	go func() {
		for result := range results {
			sub.SendData(&graphqlws.DataMessagePayload{
				Data:   result.Data,
				Errors: graphqlws.ErrorsFromGraphQLErrors(result.Errors),
			})
		}
	}()
}

func (m *subManager) removeSub(conn graphqlws.Connection, sub *graphqlws.Subscription) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	id := m.makeID(conn, sub)
	if updater, ok := m.updaters[id]; ok {
		updater.cancel()
		delete(m.updaters, id)
	}
}

func newSubManager(ctx context.Context, schema *graphql.Schema) (m *subManager) {
	return &subManager{
		ctx:      ctx,
		schema:   schema,
		inner:    graphqlws.NewSubscriptionManager(schema),
		updaters: map[string]*subUpdater{},
	}
}
