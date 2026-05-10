package db

import (
	"fmt"
	"strings"
)

func rejectDuplicateIDs(ids []string) error {
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			return fmt.Errorf("duplicate id %q", id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func deleteMissingRows(tx *SQLTx, tableName, idColumn string, ids []string) error {
	if len(ids) == 0 {
		_, err := tx.Exec("DELETE FROM " + tableName)
		return err
	}
	_, err := tx.Exec("DELETE FROM "+tableName+" WHERE "+idColumn+" NOT IN ("+questionPlaceholders(len(ids))+")", stringArgs(ids)...)
	return err
}

func questionPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	parts := make([]string, count)
	for index := range parts {
		parts[index] = "?"
	}
	return strings.Join(parts, ", ")
}

func stringArgs(values []string) []any {
	args := make([]any, len(values))
	for index, value := range values {
		args[index] = value
	}
	return args
}
