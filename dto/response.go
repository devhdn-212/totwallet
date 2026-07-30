package dto

type Response[T any] struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Record  T      `json:"record"`
}

func CreateResponseError(status int, message string) Response[string] {
	return Response[string]{
		Status:  status,
		Message: message,
		Record:  "",
	}
}
func CreateResponseErrorData(status int, message string, data map[string]string) Response[map[string]string] {
	return Response[map[string]string]{
		Status:  status,
		Message: message,
		Record:  data,
	}
}
func CreateResponseSuccess[T any](data T) Response[T] {
	return Response[T]{
		Status:  200,
		Message: "success",
		Record:  data,
	}
}

// CreateResponse dipakai kalau body "status" harus BEDA dari 200 tapi tetap butuh bawa data
// bertipe (mis. 409 duplicate transaction yang tetap perlu balikin balance terkini).
// Jangan pakai CreateResponseSuccess buat kasus ini — field status di body bakal
// tetap ke-hardcode 200 walau HTTP status code aslinya bukan 200 (respons jadi tidak konsisten).
func CreateResponse[T any](status int, message string, data T) Response[T] {
	return Response[T]{
		Status:  status,
		Message: message,
		Record:  data,
	}
}
