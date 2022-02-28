// Package gqlsub provides GraphQL subscriptions functionality.
package gqlsub

import (
	"context"
	"sync"

	"github.com/functionalfoundry/graphqlws"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
)

type updater struct {
	sub    *graphqlws.Subscription
	cancel context.CancelFunc
}

// Manager enhances graphqlws.Manager.
type Manager struct {
	ctx      context.Context
	schema   *graphql.Schema
	inner    graphqlws.SubscriptionManager
	handlers HandlerMap

	mutex    sync.Mutex
	updaters map[string]*updater
}

// Subscriptions implements graphqlws.SubscriptionManager interface.
func (m *Manager) Subscriptions() graphqlws.Subscriptions {
	return m.inner.Subscriptions()
}

// AddSubscription implements graphqlws.SubscriptionManager interface.
func (m *Manager) AddSubscription(conn graphqlws.Connection, sub *graphqlws.Subscription) []error {
	errs := m.inner.AddSubscription(conn, sub)
	if len(errs) == 0 {
		m.addSub(conn, sub)
	}
	return errs
}

// RemoveSubscription implements graphqlws.SubscriptionManager interface.
func (m *Manager) RemoveSubscription(conn graphqlws.Connection, sub *graphqlws.Subscription) {
	m.removeSub(conn, sub)
	m.inner.RemoveSubscription(conn, sub)
}

// RemoveSubscriptions implements graphqlws.SubscriptionManager interface.
func (m *Manager) RemoveSubscriptions(conn graphqlws.Connection) {
	for _, sub := range m.Subscriptions()[conn] {
		m.removeSub(conn, sub)
	}
	m.inner.RemoveSubscriptions(conn)
}

func (m *Manager) makeID(conn graphqlws.Connection, sub *graphqlws.Subscription) string {
	return conn.ID() + ":" + sub.ID
}

func (m *Manager) addSub(conn graphqlws.Connection, sub *graphqlws.Subscription) {
	if len(sub.Fields) != 1 {
		return
	}
	fieldName := sub.Fields[0]
	h := m.handlers[fieldName]
	if h == nil {
		return
	}

	id := m.makeID(conn, sub)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	ctx, cancel := context.WithCancel(m.ctx)
	m.updaters[id] = &updater{
		sub:    sub,
		cancel: cancel,
	}

	updates := make(chan interface{})
	go func() {
		for update := range updates {
			if e, ok := update.(error); ok {
				sub.SendData(&graphqlws.DataMessagePayload{
					Errors: []error{gqlerrors.FormatError(e)},
				})
				cancel()
				continue
			}
			result := graphql.Execute(graphql.ExecuteParams{
				Schema:        *m.schema,
				Root:          update,
				AST:           sub.Document,
				OperationName: sub.OperationName,
				Args:          sub.Variables,
				Context:       ctx,
			})
			sub.SendData(&graphqlws.DataMessagePayload{
				Data:   result.Data,
				Errors: graphqlws.ErrorsFromGraphQLErrors(result.Errors),
			})
		}
	}()
	go h(ctx, sub, updates)
}

func (m *Manager) removeSub(conn graphqlws.Connection, sub *graphqlws.Subscription) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	id := m.makeID(conn, sub)
	if updater, ok := m.updaters[id]; ok {
		updater.cancel()
		delete(m.updaters, id)
	}
}

// NewManager creates a Manager.
func NewManager(ctx context.Context, schema *graphql.Schema, handlers HandlerMap) (m *Manager) {
	return &Manager{
		ctx:      ctx,
		schema:   schema,
		inner:    graphqlws.NewSubscriptionManager(schema),
		handlers: handlers,
		updaters: map[string]*updater{},
	}
}
