package u

// 仅供低代码项目使用的公共方法放这

// WareHouseSchema 根据源schema和connector的server.name拼出仓库中对应的schema name
func WareHouseSchema(originalSchema, serverName string) string {
	prefix := originalSchema
	suffix := serverName
	slen := len(suffix)
	if slen > 9 {
		// serverName后13位是_UUID8()，可以截取掉
		suffix = serverName[:slen-9]
		slen = len(suffix)
	}
	plen := len(prefix)
	if plen+slen < 63 {
		return prefix + "_" + suffix
	}
	truncateLen := 63 - plen - slen
	// 因为serverName是确定小于63长度的，见dbzc的connector代码，所以一定是截短originalSchema。
	prefix = prefix[:plen-truncateLen]
	return prefix + "_" + suffix
}
