package main

import (
	"context"
	"sync"

	"github.com/usnistgov/ndn-dpdk/ndn"
)

var (
	fibLock    sync.Mutex
	fibNameGid = map[string]string{}
)

func fibToGid(name ndn.Name) (gID string) {
	fibLock.Lock()
	defer fibLock.Unlock()
	return fibNameGid[name.String()]
}

type Fib struct{}

func (Fib) List(args struct{}, reply *[]FibItem) error {
	e := client.Do(context.TODO(), `
		{
			fib {
				id
				name
			}
		}
	`, nil, "fib", reply)
	if e != nil {
		return e
	}

	fibLock.Lock()
	defer fibLock.Unlock()
	for _, item := range *reply {
		fibNameGid[item.Name.String()] = item.Gid
	}
	return nil
}

func (Fib) Insert(args FibInsertArg, reply *struct{}) error {
	var gNexthops []string
	faceLock.Lock()
	defer faceLock.Unlock()
	for _, nh := range args.Nexthops {
		gID := faceNidGid[nh]
		if gID == "" {
			return errNoFace
		}
		gNexthops = append(gNexthops, gID)
	}

	var item FibItem
	e := client.Do(context.TODO(), `
		mutation insertFibEntry($name: Name!, $nexthops: [ID!]!) {
			insertFibEntry(name: $name, nexthops: $nexthops) {
				id
				name
			}
		}
	`, map[string]interface{}{
		"name":     args.Name,
		"nexthops": gNexthops,
	}, "insertFibEntry", &item)
	if e != nil {
		return e
	}

	fibLock.Lock()
	defer fibLock.Unlock()
	fibNameGid[item.Name.String()] = item.Gid
	return nil
}

func (Fib) Erase(args FibNameArg, reply *struct{}) error {
	gID := fibToGid(args.Name)
	if gID == "" {
		return nil
	}

	e := client.Do(context.TODO(), `
		mutation delete($id: ID!) {
			delete(id: $id)
		}
	`, map[string]interface{}{
		"id": gID,
	}, "", nil)
	if e != nil {
		return e
	}

	fibLock.Lock()
	defer fibLock.Unlock()
	delete(fibNameGid, args.Name.String())
	return nil
}

func (Fib) Find(args FibNameArg, reply *FibLookupReply) error {
	reply.HasEntry = false
	gID := fibToGid(args.Name)
	if gID == "" {
		return nil
	}

	var res struct {
		Nexthops []struct {
			Nid int `json:"nid"`
		} `json:"nexthops"`
	}
	e := client.Do(context.TODO(), `
		query getFibEntry($id: ID!) {
			node(id: $id) {
				... on FibEntry {
					nexthops {
						nid
					}
				}
			}
		}
	`, map[string]interface{}{
		"id": gID,
	}, "node", &res)
	if e != nil {
		return e
	}

	reply.HasEntry = true
	reply.Name = args.Name
	faceLock.Lock()
	defer faceLock.Unlock()
	for _, nh := range res.Nexthops {
		reply.Nexthops = append(reply.Nexthops, nh.Nid)
	}
	return nil
}

type FibItem struct {
	Gid  string   `json:"id"`
	Name ndn.Name `json:"name"`
}

func (item FibItem) MarshalJSON() ([]byte, error) {
	return []byte(`"` + item.Name.String() + `"`), nil
}

type FibNameArg struct {
	Name ndn.Name
}

type FibInsertArg struct {
	Name     ndn.Name
	Nexthops []int
}

type FibLookupReply struct {
	HasEntry   bool
	Name       ndn.Name
	Nexthops   []int
	StrategyId int
}
