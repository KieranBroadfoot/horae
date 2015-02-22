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

func SetTagsForObject(uuid gocql.UUID, tags []string, typeOfObject string) {
	tagsFromDB := GetTagsForObject(uuid)
	for _, tag := range tags {
		if isStringInSlice(tag, tagsFromDB) {
			tagsFromDB = findAndRemoveInSlice(tag, tagsFromDB)
		} else {
			session.Query(`insert into tags (object_uuid, tag, type) VALUES (?, ?, ?)`, uuid, tag, typeOfObject).Exec()
		}
	}
	for _, tagToDelete := range tagsFromDB {
		session.Query(`delete from tags where object_uuid = ? and type = ? and tag = ?`, uuid, typeOfObject, tagToDelete).Exec()
	}
}

func DeleteTagsForObject(uuid gocql.UUID) error {
	if err := session.Query(`delete from tags where object_uuid = ?`, uuid).Exec(); err != nil {
		return err
	}
	return nil
}

func isStringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func findAndRemoveInSlice(a string, list []string) []string {
	for idx, b := range list {
		if b == a {
			list = append(list[:idx], list[idx+1:]...)
		}
	}
	return list
}
