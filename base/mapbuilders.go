package evercore

// Maps a slice of IdNamePair to a map of name to id.
func MapNameToId(idNamePair []IdNamePair) NameIdMap {
	nameMap := make(NameIdMap, len(idNamePair))
	for _, val := range idNamePair {
		nameMap[val.Name] = val.Id
	}
	return nameMap
}

// Maps a slice of IdNamePair to a map of id to name.
func MapIdToName(idNamePair []IdNamePair) IdNameMap {
	nameMap := make(IdNameMap, len(idNamePair))
	for _, val := range idNamePair {
		nameMap[val.Id] = val.Name
	}
	return nameMap
}
