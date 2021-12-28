package main

import "sync"

type TermStore struct {
	sync.Mutex
	terms map[string]*Term
}

var termStore = &TermStore{
	terms: map[string]*Term{},
}

func (store *TermStore) Put(term *Term) {
	store.Lock()
	defer store.Unlock()

	store.terms[term.Id] = term
}

func (store *TermStore) Get(id string) *Term {
	store.Lock()
	defer store.Unlock()

	return store.terms[id]
}

func (store *TermStore) Remove(term *Term) {
	store.Lock()
	defer store.Unlock()

	delete(store.terms, term.Id)
}

func (store *TermStore) All() []*Term {
	store.Lock()
	defer store.Unlock()

	terms := []*Term{}
	for _, term := range store.terms {
		terms = append(terms, term)
	}

	return terms
}
