// Package meta builds XMP metadata streams for PDF documents, including
// Dublin Core, PDF/A-4, and PDF/UA-2 extensions.
package meta

// MetadataStream builds an XMP metadata stream dictionary and the raw XML
// byte content from the given configuration.
func MetadataStream(config XMPConfig) (dict map[string]interface{}, data []byte) {
	data = BuildXMP(config)
	dict = map[string]interface{}{
		"/Type":    "/Metadata",
		"/Subtype": "/XML",
		"/Length":  len(data),
	}
	return
}
