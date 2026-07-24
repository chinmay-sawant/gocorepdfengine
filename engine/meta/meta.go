package meta

func MetadataStream(config XMPConfig) (dict map[string]interface{}, data []byte) {
	data = BuildXMP(config)
	dict = map[string]interface{}{
		"/Type":    "/Metadata",
		"/Subtype": "/XML",
		"/Length":  len(data),
	}
	return
}
