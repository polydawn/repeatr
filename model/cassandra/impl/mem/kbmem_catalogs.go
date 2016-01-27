package cassandra_mem

import (
	"polydawn.net/repeatr/model/catalog"
)

func (kb *Base) Catalog(id catalog.ID) *catalog.Book {
	kb.mutex.Lock()
	defer kb.mutex.Unlock()
	return kb.catalogs[id]
}

func (kb *Base) ListCatalogs() []catalog.ID {
	// This might be advised to return an iterator later.
	kb.mutex.Lock()
	defer kb.mutex.Unlock()
	ret := make([]catalog.ID, 0, len(kb.catalogs))
	for k := range kb.catalogs {
		ret = append(ret, k)
	}
	return ret
}

func (kb *Base) ObserveCatalogs(ch chan<- catalog.ID) {
	kb.mutex.Lock()
	defer kb.mutex.Unlock()
	kb.catalogObservers = append(kb.catalogObservers, ch)
}

func (kb *Base) PublishCatalog(book *catalog.Book) {
	kb.mutex.Lock()
	kb.catalogs[book.ID] = book
	var observers []chan<- catalog.ID
	copy(kb.catalogObservers, observers)
	kb.mutex.Unlock()
	for _, obvs := range observers {
		obvs <- book.ID
	}
}
