package gqlsub

import (
	"context"

	"github.com/functionalfoundry/graphqlws"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

// Handler callback is invoked when a subscription is added to the specified field.
//  ctx: a context that is canceled when the subscription is removed.
//  sub: the subscription.
//  updates: a channel for sending updates; it should be closed when Handler returns.
type Handler func(ctx context.Context, sub *graphqlws.Subscription, updates chan<- interface{})

// HandlerMap is a map from field name to Handler.
type HandlerMap map[string]Handler

// findField extracts first AST field from subscription query.
func findField(sub *graphqlws.Subscription) *ast.Field {
	odefs := []*ast.OperationDefinition{}
	for _, def := range sub.Document.Definitions {
		if odef, ok := def.(*ast.OperationDefinition); ok {
			odefs = append(odefs, odef)
		}
	}

	for _, odef := range odefs {
		if len(odefs) != 1 && odef.Name.Value != sub.OperationName {
			continue
		}

		set := odef.GetSelectionSet()
		if set == nil {
			continue
		}

		for _, sel := range set.Selections {
			if field, ok := sel.(*ast.Field); ok {
				return field
			}
		}
	}
	return nil
}

// GetArg extracts argument value from AST field.
func GetArg(sub *graphqlws.Subscription, argName string, scalar *graphql.Scalar) interface{} {
	field := findField(sub)
	if field == nil {
		return nil
	}

	for _, arg := range field.Arguments {
		if arg.Name.Value != argName {
			continue
		}

		if variable, ok := arg.Value.(*ast.Variable); ok {
			val := sub.Variables[variable.Name.Value]
			if val != nil {
				return scalar.ParseValue(val)
			}
			return nil
		}

		return scalar.ParseLiteral(arg.Value)
	}
	return nil
}
