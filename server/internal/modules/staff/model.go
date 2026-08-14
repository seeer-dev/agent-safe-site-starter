package staff

type StaffMember struct {
	ID             string `json:"id"`
	DisplayName    string `json:"display_name"`
	Email          string `json:"email"`
	RoleLabel      string `json:"role_label"`
	SupabaseUserID string `json:"supabase_user_id"`
	Status         string `json:"status"` // active|disabled
	UpdatedUnix    int64  `json:"updated_unix"`
}

type StaffMemberInput struct {
	DisplayName    string `json:"display_name"`
	Email          string `json:"email"`
	RoleLabel      string `json:"role_label"`
	SupabaseUserID string `json:"supabase_user_id"`
	Status         string `json:"status"`
}
