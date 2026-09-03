package httpx

import "net/http"

type Validator interface {
	Validate() error
}

// Validate is a generic middleware that parses JSON request bodies, validates them,
// and passes the validated data to the provided handler function.
// It returns an HTTPErrorFunc that can be wrapped with ErrorLogger for consistent error handling.
// If parsing fails, it returns a 400 Bad Request error.
// If validation fails, it returns a 400 Bad Request error with the validation error message.
//
// Example usage:
//
//	type CreateUserRequest struct {
//		Name  string `json:"name"`
//		Email string `json:"email"`
//	}
//
//	func (r CreateUserRequest) Validate() error {
//		if r.Name == "" {
//			return errors.New("name is required")
//		}
//		if r.Email == "" {
//			return errors.New("email is required")
//		}
//		return nil
//	}
//
//	func createUser(w http.ResponseWriter, r *http.Request, req CreateUserRequest) error {
//		// req is guaranteed to be valid here
//		return WriteJSON(w, http.StatusCreated, map[string]string{"id": "123"})
//	}
//
//	mux := http.HandleFunc("/users", httpx.Validate[CreateUserRequest](createUser))
//  
func Validate[V Validator](handler func(http.ResponseWriter, *http.Request, V) error) HTTPErrorFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		var v V
		if err := ParseJSON(r, &v); err != nil {
			return NewHTTPError(http.StatusBadRequest, err)
		}
		if err := v.Validate(); err != nil {
			return NewHTTPError(http.StatusBadRequest, err)
		}
		return handler(w, r, v)
	}
}
