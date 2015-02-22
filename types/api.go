package types

type Success struct {
	Message string `json:"message,required" description:"Returned status message"`
}

type Error struct {
	Code    uint32 `json:"code,required" description:"The unique identifier of the returned error"`
	Message string `json:"message,required" description:"An error message"`
}
