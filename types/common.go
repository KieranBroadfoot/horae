package types

import (
	"github.com/gocql/gocql"
)

func GetTagsForObject(uuid gocql.UUID) []string {
	tags := []string{}
	tag := ""
	iteration := session.Query("select tag from tags where object_uuid = ?", uuid).Iter()
	for iteration.Scan(&tag) {
		tags = append(tags, tag)
	}
	return tags
}

func SetTagsForObject(uuid gocql.UUID, tags []string, typeOfObject string) error {
	for _, tag := range tags {
		if err := session.Query(`insert into tags (object_uuid, tag, type) VALUES (?, ?, ?)`, uuid, tag, typeOfObject).Exec(); err != nil {
			return err
		}
	}
	return nil
}
