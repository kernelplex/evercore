package evercore

func MapNameToId(idNamePair []IdNamePair) NameIdMap {
	nameMap := make(NameIdMap, len(idNamePair))
	for _, val := range idNamePair {
		nameMap[val.Name] = val.Id
	}
	return nameMap
}

func MapIdToName(idNamePair []IdNamePair) IdNameMap {
	nameMap := make(IdNameMap, len(idNamePair))
	for _, val := range idNamePair {
		nameMap[val.Id] = val.Name
	}
	return nameMap
}
