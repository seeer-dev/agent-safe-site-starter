# Auth

Supabase Auth authenticates production users; Go authorizes actions.

Browser sends the Supabase access token to Go. Go turns it into explicit `auth.Principal`. Services receive Principal as a normal argument when authorization matters.

Do not query application tables directly from the browser via Supabase. Do not move business authorization into frontend route guards.
