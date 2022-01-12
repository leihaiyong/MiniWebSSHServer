package main

import (
	"errors"
	"sync"
)

type TermStore struct {
	sync.Mutex
	terms map[string]*Term
}

var termStore = &TermStore{
	terms: map[string]*Term{},
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

func (store *TermStore) Add(term *Term) {
	store.Lock()
	defer store.Unlock()

	store.terms[term.Id] = term
}

func (store *TermStore) Del(term *Term, close bool) {
	store.Lock()
	defer store.Unlock()

	delete(store.terms, term.Id)

	if close {
		term.Close()
	}
}

func (store *TermStore) Get(id string) (*Term, error) {
	store.Lock()
	defer store.Unlock()

	term := store.terms[id]
	if term == nil {
		return nil, errors.New("term " + id + " not exist")
	}

	return term, nil
}

func (store *TermStore) Do(id string, handler func(term *Term) error) (*Term, error) {
	store.Lock()
	defer store.Unlock()

	term := store.terms[id]
	if term == nil {
		return nil, errors.New("term " + id + " not exist")
	}

	return term, handler(term)
}
