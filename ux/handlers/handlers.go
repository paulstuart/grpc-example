package handlers

import (
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/paulstuart/grpc-example/ux/client"
	"go.opentelemetry.io/otel"
)

// Handler manages HTTP requests
type Handler struct {
	client    *client.Client
	templates *template.Template
}

// NewHandler creates a new handler
func NewHandler(apiClient *client.Client, templates *template.Template) *Handler {
	return &Handler{
		client:    apiClient,
		templates: templates,
	}
}

// IntToRole converts role int to string
// TODO: have the DB be the source of truth for these mappings?
func IntToRole(role int32) string {
	switch role {
	case 0:
		return "GUEST"
	case 1:
		return "MEMBER"
	case 2:
		return "ADMIN"
	case 3:
		return "MODERATOR"
	default:
		return "GUEST"
	}
}

// IntToStatus converts status int to string
func IntToStatus(status int32) string {
	switch status {
	case 0:
		return "INACTIVE"
	case 1:
		return "ACTIVE"
	case 2:
		return "SUSPENDED"
	case 3:
		return "DELETED"
	default:
		return "INACTIVE"
	}
}

// Home displays the main page
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "User Management",
	}
	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ListUsers returns the users list
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.client.ListUsers()
	if err != nil {
		log.Printf("list users error: %v", err)
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Users": users,
	}
	if err := h.templates.ExecuteTemplate(w, "users-list.html", data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// GetUser returns a single user
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {

	ctx, span := otel.Tracer("frontend-tracer").Start(r.Context(), "call-backend-handler")
	defer span.End()
	_ = ctx

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := h.client.GetUser(uint32(id))
	if err != nil {
		log.Printf("get user error: %v", err)
		http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		return
	}

	// Convert string role/status to int for form
	user.RoleInt = client.RoleToInt(user.Role)
	user.StatusInt = client.StatusToInt(user.Status)

	data := map[string]interface{}{
		"User": user,
	}
	if err := h.templates.ExecuteTemplate(w, "user-detail.html", data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// NewUserForm shows the form to create a new user
func (h *Handler) NewUserForm(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.ExecuteTemplate(w, "user-form.html", nil); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// CreateUser handles user creation
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	id, _ := strconv.ParseUint(r.FormValue("id"), 10, 32)
	role, _ := strconv.ParseInt(r.FormValue("role"), 10, 32)
	status, _ := strconv.ParseInt(r.FormValue("status"), 10, 32)

	user := &client.User{
		ID:       uint32(id),
		Username: r.FormValue("username"),
		Email:    r.FormValue("email"),
		Phone:    r.FormValue("phone"),
		Role:     IntToRole(int32(role)),
		Status:   IntToStatus(int32(status)),
	}

	if displayName := r.FormValue("displayName"); displayName != "" {
		user.Profile = &client.Profile{
			DisplayName: displayName,
			Bio:         r.FormValue("bio"),
		}
	}

	if err := h.client.CreateUser(user); err != nil {
		log.Printf("create user error: %v", err)
		// Send error message back to the user
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div style="background-color: #fee; border: 1px solid #fcc; padding: 1rem; margin: 1rem 0; border-radius: 4px;">
			<strong style="color: #c00;">Error creating user:</strong><br>
			` + err.Error() + `
			<br><small>Note: User IDs and usernames must be unique.</small>
		</div>`))
		return
	}

	// On success, close the modal and refresh the list
	w.Header().Set("HX-Trigger", "userCreated")
	w.Header().Add("HX-Trigger", "closeModal")
	h.ListUsers(w, r)
}

// EditUserForm shows the form to edit a user
func (h *Handler) EditUserForm(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := h.client.GetUser(uint32(id))
	if err != nil {
		log.Printf("get user error: %v", err)
		http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		return
	}

	// Convert string role/status to int for form
	user.RoleInt = client.RoleToInt(user.Role)
	user.StatusInt = client.StatusToInt(user.Status)

	data := map[string]interface{}{
		"User": user,
	}
	if err := h.templates.ExecuteTemplate(w, "user-edit.html", data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// UpdateUser handles user updates
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	role, _ := strconv.ParseInt(r.FormValue("role"), 10, 32)
	status, _ := strconv.ParseInt(r.FormValue("status"), 10, 32)

	user := &client.User{
		ID:       uint32(id),
		Username: r.FormValue("username"),
		Email:    r.FormValue("email"),
		Phone:    r.FormValue("phone"),
		Role:     IntToRole(int32(role)),
		Status:   IntToStatus(int32(status)),
	}

	if displayName := r.FormValue("displayName"); displayName != "" {
		user.Profile = &client.Profile{
			DisplayName: displayName,
			Bio:         r.FormValue("bio"),
		}
	}

	paths := []string{"username", "email", "phone", "role", "status", "profile"}
	_, err = h.client.UpdateUser(user, paths)
	if err != nil {
		log.Printf("update user error: %v", err)
		// Send error message back to the user
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div style="background-color: #fee; border: 1px solid #fcc; padding: 1rem; margin: 1rem 0; border-radius: 4px;">
			<strong style="color: #c00;">Error updating user:</strong><br>
			` + err.Error() + `
			<br><small>Note: Usernames must be unique. Try a different username.</small>
		</div>`))
		return
	}

	// On success, close the modal and refresh the list
	w.Header().Set("HX-Trigger", "userUpdated")
	// Also trigger closing the modal
	w.Header().Add("HX-Trigger", "closeModal")
	h.ListUsers(w, r)
}

// DeleteUser handles user deletion
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := h.client.DeleteUser(uint32(id)); err != nil {
		log.Printf("delete user error: %v", err)
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", "userDeleted")
	h.ListUsers(w, r)
}

// FilterByRole filters users by role
func (h *Handler) FilterByRole(w http.ResponseWriter, r *http.Request) {
	roleStr := r.URL.Query().Get("role")
	if roleStr == "" || roleStr == "all" {
		h.ListUsers(w, r)
		return
	}

	role, err := strconv.ParseInt(roleStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}

	users, err := h.client.ListUsersByRole(int32(role))
	if err != nil {
		log.Printf("filter by role error: %v", err)
		http.Error(w, "Failed to filter users", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Users": users,
	}
	if err := h.templates.ExecuteTemplate(w, "users-list.html", data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
