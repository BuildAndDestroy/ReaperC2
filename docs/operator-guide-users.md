# Users

**Path:** `/users`  
**Requires:** Admin role

Create portal accounts and assign **Admin** or **Operator**. Change an existing user’s role with the **Role** dropdown in the accounts table (admins only; you cannot change your own role). Enable/disable accounts (disabled users cannot sign in). Admins cannot disable themselves from the list actions.

API: `POST /api/users`, `PATCH /api/users/{username}` with `{ "role": "admin" | "operator" }` and/or `{ "disabled": true | false }`. Operators do not see this page.
