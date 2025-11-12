# CRUD UI Guideline

This guideline standardizes how CRUD experiences are delivered in the admin web app. It complements the ShadCN-first rules and applies to every new domain (e.g., users, schema categories, future resources).

## 1. General Principles

1. **ShadCN-first:** Prefer blocks/components from the shadcn registry. Only assemble custom JSX by composing registered primitives.
2. **Consistency:** Table layouts, forms, dialogs, and toasts must behave uniformly across domains.
3. **Accessibility:** Favor Radix-backed components; ensure keyboard/focus support is preserved.
4. **Feedback:** Use the shared toast system (`sonner`) for success/error. Avoid `alert()` or bespoke banners.

## 2. Listing Page Requirements

1. **Table Layout**
   - Use the shadcn Data Table block (TanStack Table) with:
     - Row selection (checkbox column).
     - Default visible columns for key attributes.
     - Optional columns toggled via “Customize columns”.
   - Add actions column with edit/delete buttons (icon + label), right-aligned.
2. **Column Controls**
   - “Customize columns” menu must include optional fields (e.g., IDs, timestamps, descriptions).
   - Persisted visibility per session; follow existing pattern (local state).
3. **Toolbar**
   - Left side reserved for filters/search (even if empty initially).
   - Right side: “Customize columns” button + “Add {Resource}” button.
4. **Pagination**
   - Integrate pagination controls from shadcn.
   - Show `Rows per page`, current page, and next/prev navigation.
5. **Empty/Error States**
   - Empty: single row with “No {resourcePlural} found.”
   - Error: inline alert or toast with retry guidance; prefer showing a bordered alert panel.
6. **Delete Action**
   - Must confirm via shadcn alert dialog.
   - On success, toast using success variant; on failure, show error toast with message.
7. **Bulk Actions (optional)**
   - If row selection is present, future bulk operations should fit into the toolbar (but can be omitted initially).

## 3. Form Pages (Create & Edit)

1. **Layout**
   - Wrap content in shadcn `Card` with header/title and supporting description.
   - Provide “Cancel” (ghost) button top-right linking back to list.
2. **Form Components**
   - Use `@/components/ui/form` helpers (Form, FormField, FormItem, etc.).
   - Inputs: `Input`, `Textarea`, `Select`, etc. from `@/components/ui/*`.
   - Limit custom styling; rely on design tokens.
3. **Validation**
   - Use Zod + React Hook Form.
   - Display errors via `FormMessage`.
4. **Submission**
   - Submit button text: “Save {resource}” (create) / “Save changes” (edit).
   - Disable button when pending.
   - After success, redirect to list with toast.
5. **Loading/Error States**
   - Edit page: show skeletons while fetching existing data.
   - Error fetching: inline destructive alert (“Unable to load {resource}”).

## 4. Routing & Hooks

1. **Routes**
   - List: `/{{resource}}`
   - Create: `/{{resource}}/new`
   - Edit: `/{{resource}}/:id/edit`
2. **Hooks**
   - Provide React Query hooks for list/get/create/update/delete under the domain folder.
   - Hooks should invalidate relevant query keys after mutations.

## 5. Backend Alignment

1. **Payloads**
   - Create and update forms must align with OpenAPI schemas.
   - Optional fields allowed on update should map 1:1 with handler input.

## 6. Testing & Validation

1. **Backend**
   - Add service tests covering fields updates, validation errors, and success paths.
   - Update handler tests to assert payload mapping and success responses.
2. **Frontend**
   - Add Vitest tests as needed.
   - Manually verify in development:
     - List loads data.
     - Create -> list shows toast.
     - Edit updates fields.
     - Delete removes row with toast.

## 7. Reuse Checklist

- [ ] Have you checked existing components before building new ones?
- [ ] Are all newly introduced UI primitives sourced from shadcn?
- [ ] Are toast notifications consistent with other domains?
- [ ] Do forms share the common `Form` helper and follow the same layout?
- [ ] Do tables include selection, column toggles, pagination, and delete actions?

This guideline should be referenced before starting any CRUD implementation to ensure consistency and rapid delivery.  
