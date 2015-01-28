package indexer

import (
	"sort"
	"github.com/ctdk/goiardi/datastore"
	"github.com/ctdk/goas/v2/logger"
	"strings"
	"database/sql"
	"regexp"
)

type PostgreSQLIndex struct {
}


func (i *PostgreSQLIndex) createCollection(idxName string) {
	// nothing to do here
}

func (i *PostgreSQLIndex) deleteCollection(idxName string) {
	logger.Debugf("Deleting index: %s", idxName)

	switch idxName {
		case "node":
			i.deleteIndex("goiardi.node_search")
		default:
			logger.Errorf("unkown index: %s", idxName)
	}

}

func (i *PostgreSQLIndex) saveIndex(object Indexable) {
	switch object.Index() {
		case "node":
			i.indexNode(object)
		default:
			logger.Errorf("unkown index: %s", object.Index())
	}
}

func (i *PostgreSQLIndex) deleteItem(idxName string, doc string) error {
	switch idxName {
		case "node":
			params := make([]interface{}, 1)
			params[0] = doc
			return i.runInTransaction("DELETE FROM node_search WHERE name = $1", params)
		default:
			logger.Errorf("unkown index: %s", idxName)
	}

	return nil
}

func (i *PostgreSQLIndex) search(idx string, term string, notop bool) (map[string]*Document, error) {
	parts := strings.SplitN(term, ":", 2)

	if parts[0] == "*" {
		parts[0] = "%"
	}

	r := regexp.MustCompile(`\*|\?`)

	value := parts[1]
	isWildcard := r.MatchString(value)
	isPrefix := false
	if isWildcard {
		position := r.FindStringIndex(value)

		isPrefix = position[1] == len(value)
		if isPrefix {
			//prefix search
			value = value[0:len(value) - 1] + ":*"
		} else {
			value = value[0:position[1]]
		}
	}

	sqlNot := ""
	if notop {
		sqlNot = "NOT"
	}

	results := make(map[string]*Document)

	switch idx {
		case "node":
			sqlStmt := "SELECT ns.name, ns.value FROM goiardi.node_search ns WHERE ns.field LIKE $1 AND " + sqlNot + " ns.value @@ $2::tsquery"
			params := make([]interface{}, 2)
			params[0] = parts[0]
			params[1] = value

			values, err := queryPostgresIndex(sqlStmt, params)

			if err != nil {
				return nil, err
			}

			return filterSearchResults(values, parts[1], isWildcard, isPrefix, notop)
		default:
			logger.Errorf("unkown index: %s", idx)
	}

	return results, nil
}

func (i *PostgreSQLIndex) searchText(idx string, term string, notop bool) (map[string]*Document, error) {
	results := make(map[string]*Document)

	return results, nil
}

func (i *PostgreSQLIndex) searchRange(idx string, field string, start string, end string, inclusive bool) (map[string]*Document, error) {
	results := make(map[string]*Document)

	return results, nil
}

func (i *PostgreSQLIndex) endpoints() []string {
	endpoints := []string{"client", "environment", "node", "role"}

	// todo query databag names later from index

	sort.Strings(endpoints)
	return endpoints
}

func (i *PostgreSQLIndex) clear() {
	for _, index := range i.endpoints() {
		i.deleteCollection(index)
	}
}

func (i *PostgreSQLIndex) makeDefaultCollections() {
	// nothing to do here
}

func (i *PostgreSQLIndex) initialize() {
	// nothing to do here
}

func (i *PostgreSQLIndex) save() error {
	// nothing to do here, every operation is saved instantly
	return nil
}

func (i *PostgreSQLIndex) load() error {
	// nothing to do here, every operation is saved instantly
	return nil
}

func (i *PostgreSQLIndex) indexNode(object Indexable) {
	fields := object.Flatten()

	logger.Debugf("Inserting document into postgres node index: %s", object.DocID())

	tx, nil := datastore.Dbh.Begin()
	_, err := tx.Exec("DELETE FROM goiardi.node_search WHERE node_search.name = $1", object.DocID())
	if err != nil {
		tx.Rollback()
		return
	}

	for field_name, value := range fields {
		switch value.(type) {
			case []string:
				value = strings.Join(value.([]string), " ")
			default:
		}
		_, err := tx.Exec("INSERT INTO goiardi.node_search (name, field, value) VALUES($1, $2, $3)", object.DocID(), field_name, value)
		if err != nil {
			tx.Rollback()
			logger.Errorf("Error inserting into postgres node index: %s", err)
			return
		}
	}

	tx.Commit()
	logger.Debugf("Inserted document into postgres node index: %s", object.DocID())
}

func (i *PostgreSQLIndex) deleteIndex(tableName string) error {
	return i.runInTransaction("DELETE FROM " + tableName, make([]interface{}, 0))
}

func (i *PostgreSQLIndex) runInTransaction(query string, params []interface{}) error {
	tx, nil := datastore.Dbh.Begin()
	_, err := tx.Exec(query, params...)
	if err != nil {
		tx.Rollback()
		logger.Errorf("Failed to run query: %s %s", query, err)
		return err
	}
	return tx.Commit()
}

func queryPostgresIndex(sqlStmt string, params []interface{}) (map[string]string, error) {
	results := make(map[string]string)

	stmt, err := datastore.Dbh.Prepare(sqlStmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	queryParams := make([]interface{}, len(params))
	copy(queryParams, params[0:])

	rows, qerr := stmt.Query(queryParams...)
	defer rows.Close()

	if qerr != nil {
		if qerr == sql.ErrNoRows {
			return results, nil
		}
		return nil, qerr
	}
	for rows.Next() {
		var name string
		var value string

		err := rows.Scan(&name, &value)
		if err != nil {
			return nil, err
		}

		results[name] = value
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func filterSearchResults(values map[string]string, searchTerm string, isWildcard bool, isPrefix bool, notop bool) (map[string]*Document, error) {
	results := make(map[string]*Document)

	if (isWildcard && isPrefix) || !isWildcard {
		for name, _ := range values {
			results[name] = new(Document)
		}
	} else {
		errCh := make(chan error, len(values))
		resCh := make(chan *string, len(values))
		for name, value := range values {
			go func(name string, value string) {
				term := searchTerm
				term = strings.Replace(term, "*", ".*", -1)
				term = strings.Replace(term, "?", ".{1,1}", -1)
				reComp, err := regexp.Compile(term)

				if err != nil {
					errCh <- err
					resCh <- nil
				} else {
					errCh <- nil

					match := reComp.MatchString(value)
					if (match && !notop) || (!match && notop) {
						resCh <- &name
					} else {
						resCh <- nil
					}
				}
			}(name, value)
		}
		for i := 0; i < len(values); i++ {
			e := <-errCh
			if e != nil {
				return nil, e
			}
		}
		for i := 0; i < len(values); i++ {
			name := <-resCh
			if name != nil {
				results[*name] = new(Document)
			}
		}
	}

	return results, nil
}