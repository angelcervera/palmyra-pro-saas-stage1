import { Routes, Route } from 'react-router-dom'
import DashboardPage from '@/app/dashboard/page'
import UsersPage from '@/app/users/page'
import UsersCreatePage from '@/app/users/new/page'
import SchemaCategoriesPage from '@/app/schema-categories/page'
import SchemaCategoriesCreatePage from '@/app/schema-categories/new/page'
import SchemaCategoriesEditPage from '@/app/schema-categories/edit/page'
import SchemaRepositoryPage from '@/app/schema-repository/page'
import SchemaRepositoryCreatePage from '@/app/schema-repository/new/page'
import SchemaVersionDetailPage from '@/app/schema-repository/version-detail/page'
import SchemaVersionEditPage from '@/app/schema-repository/edit/page'
import EntitiesPage from '@/app/entities/page'
import NotFoundPage from '@/app/not-found'
import { AdminLayout } from '@/app/admin-layout'

// Router skeleton for the admin app.
// NOTE: Add new domain routes here as they become available.
export function AdminRoutes() {
  return (
    <Routes>
      <Route element={<AdminLayout />}>
        <Route index element={<DashboardPage />} />
        <Route path="/users" element={<UsersPage />} />
        <Route path="/users/new" element={<UsersCreatePage />} />
        <Route path="/schema-categories" element={<SchemaCategoriesPage />} />
        <Route path="/schema-categories/new" element={<SchemaCategoriesCreatePage />} />
        <Route path="/schema-categories/:categoryId/edit" element={<SchemaCategoriesEditPage />} />
        <Route path="/schema-repository" element={<SchemaRepositoryPage />} />
        <Route path="/schema-repository/new" element={<SchemaRepositoryCreatePage />} />
        <Route
          path="/schema-repository/:schemaId/versions/:schemaVersion"
          element={<SchemaVersionDetailPage />}
        />
        <Route
          path="/schema-repository/:schemaId/versions/:schemaVersion/edit"
          element={<SchemaVersionEditPage />}
        />
        <Route path="/entities" element={<EntitiesPage />} />
        {/* Not Found */}
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  )
}
