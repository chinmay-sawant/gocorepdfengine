package meta

func MetadataStream(config XMPConfig) (map[string]interface{}, []byte) {
	data := BuildXMP(config)
	dict := map[string]interface{}{
		"/Type":    "/Metadata",
		"/Subtype": "/XML",
		"/Length":  len(data),
	}
	return dict, data
}
