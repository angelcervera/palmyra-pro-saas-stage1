import { useEffect, useMemo } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { FieldGroup, Fieldset, FieldsetDescription, FieldsetLegend } from "@/components/ui/field"
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"

const SLUG_REGEX = /^[a-z0-9]+(?:-[a-z0-9]+)*$/
const TABLE_NAME_REGEX = /^[a-z][a-z0-9_]*$/
const DEFAULT_SCHEMA_TEMPLATE = `{
  "title": "New schema version",
  "type": "object",
  "properties": {}
}`

const schemaVersionFormSchema = z.object({
  tableName: z
    .string()
    .min(1, "Table name is required")
    .regex(TABLE_NAME_REGEX, "Use lowercase snake_case starting with a letter")
    .transform((value) => value.trim()),
  slug: z
    .string()
    .min(1, "Slug is required")
    .regex(SLUG_REGEX, "Use lowercase letters, numbers, and hyphens")
    .transform((value) => value.trim()),
  categoryId: z.string().uuid("Select a category"),
  schemaDefinition: z
    .string()
    .min(2, "Schema definition is required")
    .superRefine((value, ctx) => {
      try {
        const parsed = JSON.parse(value)
        if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: "Schema definition must be a JSON object",
          })
        }
      } catch {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: "Schema definition must be valid JSON",
        })
      }
    }),
})

export type SchemaVersionFormValues = z.infer<typeof schemaVersionFormSchema>

export type SchemaVersionSubmitPayload = {
  tableName: string
  slug: string
  categoryId: string
  schemaDefinition: Record<string, unknown>
}

type ReadOnlyFields = {
  tableName?: boolean
  slug?: boolean
  category?: boolean
}

type SchemaVersionFormProps = {
  defaultValues?: Partial<SchemaVersionFormValues>
  categories?: Array<{ id: string; name: string }>
  isSubmitting?: boolean
  onSubmit: (values: SchemaVersionSubmitPayload) => Promise<void> | void
  className?: string
  readOnlyFields?: ReadOnlyFields
  submitLabel?: string
}

export function SchemaVersionForm({
  defaultValues,
  categories = [],
  isSubmitting = false,
  onSubmit,
  className,
  readOnlyFields,
  submitLabel = "Save schema version",
}: SchemaVersionFormProps) {
  const mergedDefaults = useMemo<SchemaVersionFormValues>(
    () => ({
      tableName: defaultValues?.tableName ?? "",
      slug: defaultValues?.slug ?? "",
      categoryId: defaultValues?.categoryId ?? "",
      schemaDefinition: defaultValues?.schemaDefinition ?? DEFAULT_SCHEMA_TEMPLATE,
    }),
    [defaultValues],
  )

  const form = useForm<SchemaVersionFormValues>({
    resolver: zodResolver(schemaVersionFormSchema),
    defaultValues: mergedDefaults,
    mode: "onSubmit",
  })

  useEffect(() => {
    form.reset(mergedDefaults)
  }, [form, mergedDefaults])

  const tableNameValue = form.watch("tableName")
  const slugValue = form.watch("slug")
  const slugIsDirty = Boolean(form.formState.dirtyFields.slug)

  useEffect(() => {
    if (readOnlyFields?.slug) {
      return
    }
    if (slugIsDirty) {
      return
    }
    if (!tableNameValue) {
      return
    }
    const candidate = slugify(tableNameValue)
    if (!slugValue || candidate.startsWith(slugValue)) {
      form.setValue("slug", candidate, { shouldDirty: false, shouldTouch: false, shouldValidate: false })
    }
  }, [form, readOnlyFields?.slug, slugIsDirty, slugValue, tableNameValue])

  const handleSubmit = form.handleSubmit(
    async (values) => {
      try {
        const parsed = JSON.parse(values.schemaDefinition) as Record<string, unknown>
        if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
          throw new Error("Schema definition must be a JSON object")
        }

        await onSubmit({
          tableName: values.tableName,
          slug: values.slug,
          categoryId: values.categoryId,
          schemaDefinition: parsed,
        })
      } catch (error) {
        const message = error instanceof Error ? error.message : "Schema definition must be valid JSON"
        form.setError("schemaDefinition", { type: "manual", message })
        form.setFocus("schemaDefinition")
      }
    },
    (submitErrors) => {
      const firstErrorKey = Object.keys(submitErrors)[0]
      if (firstErrorKey) {
        form.setFocus(firstErrorKey as keyof SchemaVersionFormValues)
      }
    },
  )

  return (
    <Form {...form}>
      <form
        noValidate
        className={cn("grid gap-8", className)}
        onSubmit={handleSubmit}
      >
        <Fieldset>
          <FieldsetLegend>Identifiers</FieldsetLegend>
          <FieldsetDescription>
            Set the immutable identifiers for this schema version.
          </FieldsetDescription>
          <FieldGroup>
            <FormField
              control={form.control}
              name="tableName"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Table name</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="cards_entities"
                      autoComplete="off"
                      disabled={Boolean(readOnlyFields?.tableName)}
                      readOnly={Boolean(readOnlyFields?.tableName)}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    Lowercase snake_case PostgreSQL table name.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="slug"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Slug</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="cards-schema"
                      autoComplete="off"
                      disabled={Boolean(readOnlyFields?.slug)}
                      readOnly={Boolean(readOnlyFields?.slug)}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    Kebab-case identifier exposed over the API.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </FieldGroup>
          <FormField
            control={form.control}
            name="categoryId"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Category</FormLabel>
                <Select
                  value={field.value || undefined}
                  onValueChange={field.onChange}
                  disabled={categories.length === 0 || Boolean(readOnlyFields?.category)}
                >
                  <FormControl>
                    <SelectTrigger
                      ref={field.ref}
                      onBlur={field.onBlur}
                      className="w-full"
                    >
                      <SelectValue placeholder="Select a category" />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    {categories.map((category) => (
                      <SelectItem key={category.id} value={category.id}>
                        {category.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <FormDescription>
                  Categories help group schema versions by domain.
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </Fieldset>

        <Fieldset>
          <FieldsetLegend>Schema document</FieldsetLegend>
          <FieldsetDescription>
            Paste the JSON Schema that defines this entity version.
          </FieldsetDescription>
          <FormField
            control={form.control}
            name="schemaDefinition"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Schema definition</FormLabel>
                <FormControl>
                  <Textarea
                    {...field}
                    className="font-mono text-sm"
                    rows={14}
                    spellCheck={false}
                    placeholder='{
  "title": "Schema title",
  "type": "object",
  "properties": {}
}'
                  />
                </FormControl>
                <FormDescription>
                  Provide a valid JSON Schema object. This value is persisted as-is.
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </Fieldset>

        <div className="flex justify-end">
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? "Saving…" : submitLabel}
          </Button>
        </div>
      </form>
    </Form>
  )
}

function slugify(input: string) {
  return input
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9\s-]/g, "")
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-")
}
