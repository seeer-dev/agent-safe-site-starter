# AC-013 security review — comment moderation fails closed

## Verified

- Public `GET /api/products/{slug}/comments` returns only
  `status=approved` comments plus the server-computed summary
  (count/avg/distribution). Pending and rejected rows are filtered at
  the SQL layer (`ListApprovedComments`), not hidden client-side.
- `POST /api/products/{slug}/comments` always creates `pending` —
  clients cannot self-approve; there is no public status field in
  `CommentInput`.
- Seed data proves the state machine: 3 approved comments render on the
  product page; the 4th (pending, 老茶客 on high-mountain-oolong) is
  absent from the public list but present in `/admin/comments`.
- Moderation `PATCH /api/admin/comments/{id}` requires
  `twcommerce.update`; the admin sends `{status, reply}` — the reply
  becomes `admin_reply` shown under the comment.
