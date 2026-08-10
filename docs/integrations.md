# Integrations

## Supabase Auth

The browser obtains a Supabase Auth access token. The browser sends it to the Go API as `Authorization: Bearer <token>`. The sample verifier asks Supabase Auth for the current user, then creates an explicit `auth.Principal`.

Business authorization remains in Go. The frontend never gains direct database authority.

## Cloudflare R2

Protected API endpoint `POST /api/media/presign` returns a short-lived presigned PUT URL. The browser uploads the bytes directly to R2, so Railway does not relay large files.

## Resend

The contact module stores an inquiry first, then sends a notification when `CONTACT_NOTIFY_TO` and `RESEND_API_KEY` are configured. Without a Resend key, the mail adapter logs messages locally.
