package types

type Success struct {
	message string `json:"message,required" description:"Returned status message"`
}

type Error struct {
	code    uint32 `json:"code,required" description:"The unique identifier of the returned error"`
	message string `json:"message,required" description:"An error message"`
	fields  string `json:"fields,required" description:"The fields in the originating request which are problematic"`
}
