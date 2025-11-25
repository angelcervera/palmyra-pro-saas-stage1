import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'

import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

export const schemaCategoryFormSchema = z.object({
  name: z.string().min(1, 'Name is required').max(128, 'Name must be 128 characters or fewer'),
  slug: z.string().min(1, 'Slug is required').max(128, 'Slug must be 128 characters or fewer'),
  description: z.string().max(512, 'Description must be 512 characters or fewer').optional(),
})

export type SchemaCategoryFormValues = z.infer<typeof schemaCategoryFormSchema>

type SchemaCategoryFormProps = {
  defaultValues?: Partial<SchemaCategoryFormValues>
  isSubmitting?: boolean
  onSubmit: (values: SchemaCategoryFormValues) => Promise<void> | void
  className?: string
}

export function SchemaCategoryForm({
  defaultValues,
  isSubmitting = false,
  onSubmit,
  className,
}: SchemaCategoryFormProps) {
  const form = useForm<SchemaCategoryFormValues>({
    resolver: zodResolver(schemaCategoryFormSchema),
    defaultValues: {
      name: defaultValues?.name ?? '',
      slug: defaultValues?.slug ?? '',
      description: defaultValues?.description ?? '',
    },
    mode: 'onSubmit',
    reValidateMode: 'onChange',
  })

  const errors = form.formState.errors

  const nameValue = form.watch('name')
  const slugValue = form.watch('slug')

  useEffect(() => {
    if (!slugValue) {
      const nextSlug = slugify(nameValue ?? '')
      if (nextSlug) {
        form.setValue('slug', nextSlug, { shouldDirty: true })
      }
    }
  }, [form, nameValue, slugValue])

  const submitHandler = form.handleSubmit(
    async (values) => {
      await onSubmit({
        name: values.name.trim(),
        slug: values.slug.trim(),
        description: values.description?.trim() ? values.description.trim() : undefined,
      })
    },
    (submitErrors) => {
      const firstErrorKey = Object.keys(submitErrors)[0]
      if (firstErrorKey) {
        form.setFocus(firstErrorKey as keyof SchemaCategoryFormValues)
      }
    },
  )

  return (
    <form
      className={cn('space-y-6', className)}
      onSubmit={submitHandler}
      noValidate
    >
      <div className="grid gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="category-name">Name</Label>
          <Input
            id="category-name"
            autoComplete="off"
            placeholder="Content ingestion"
            aria-invalid={Boolean(errors.name)}
            {...form.register('name')}
          />
          {errors.name && (
            <p className="text-xs font-medium text-destructive" role="alert">
              {errors.name.message}
            </p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="category-slug">Slug</Label>
          <Input
            id="category-slug"
            autoComplete="off"
            placeholder="content-ingestion"
            aria-invalid={Boolean(errors.slug)}
            {...form.register('slug')}
          />
          {errors.slug && (
            <p className="text-xs font-medium text-destructive" role="alert">
              {errors.slug.message}
            </p>
          )}
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="category-description">Description</Label>
        <Textarea
          id="category-description"
          className="min-h-[120px]"
          placeholder="Optional summary to help teammates choose the category."
          aria-invalid={Boolean(errors.description)}
          {...form.register('description')}
        />
        {errors.description && (
          <p className="text-xs font-medium text-destructive" role="alert">
            {errors.description.message}
          </p>
        )}
      </div>

      <div className="flex items-center justify-end gap-2">
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? 'Saving…' : 'Save category'}
        </Button>
      </div>
    </form>
  )
}

function slugify(input: string) {
  return input
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9\s-]/g, '')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
}
