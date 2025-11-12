import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { zodResolver } from "@hookform/resolvers/zod"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { toast } from "sonner"
import { useQueryClient } from "@tanstack/react-query"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form"
import { useCreateUser } from "../use-users"

const createUserSchema = z.object({
  email: z
    .string()
    .min(1, "Email is required")
    .email("Enter a valid email")
    .transform((value) => value.trim().toLowerCase()),
  fullName: z
    .string()
    .min(1, "Full name is required")
    .transform((value) => value.trim()),
})

type FormValues = z.infer<typeof createUserSchema>

type ProblemDetails = {
  title?: string
  detail?: string
  status?: number
  errors?: Record<string, unknown>
}

const DEFAULT_VALUES: FormValues = {
  email: "",
  fullName: "",
}

function extractProblemDetails(error: unknown): ProblemDetails | null {
  if (!error || typeof error !== "object") {
    return null
  }

  const maybe = error as Record<string, unknown>
  const hasAnyKnownKey = ["title", "detail", "status", "errors"].some((key) => key in maybe)
  if (!hasAnyKnownKey) {
    return null
  }

  const errors = typeof maybe.errors === "object" && maybe.errors !== null ? (maybe.errors as Record<string, unknown>) : undefined

  return {
    title: typeof maybe.title === "string" ? maybe.title : undefined,
    detail: typeof maybe.detail === "string" ? maybe.detail : undefined,
    status: typeof maybe.status === "number" ? maybe.status : undefined,
    errors,
  }
}

export default function CreateUserPage() {
  const navigate = useNavigate()
  const [submitError, setSubmitError] = useState<string | null>(null)
  const createUser = useCreateUser()
  const queryClient = useQueryClient()

  const form = useForm<FormValues>({
    resolver: zodResolver(createUserSchema),
    defaultValues: DEFAULT_VALUES,
    mode: "onSubmit",
  })

  const isSubmitting = createUser.isPending

  const assignServerError = (field: keyof FormValues, message: string) => {
    form.setError(field, { type: "server", message })
  }

  // Field errors are rendered via <FormMessage /> with shadcn Form primitives

  const onSubmit = form.handleSubmit(async (values) => {
    setSubmitError(null)

    try {
      await createUser.mutateAsync(values)
      await queryClient.invalidateQueries({
        predicate: (query) => Array.isArray(query.queryKey) && query.queryKey[0] === "users",
      })
      await queryClient.refetchQueries({
        predicate: (query) => Array.isArray(query.queryKey) && query.queryKey[0] === "users",
        type: "all",
      })
      form.reset(DEFAULT_VALUES)
      navigate("/users", {
        replace: true,
        state: {
          toast: {
            type: "success",
            message: "User created successfully",
          },
        },
      })
    } catch (err) {
      const problem = extractProblemDetails(err)
      if (problem?.errors) {
        for (const [key, value] of Object.entries(problem.errors)) {
          if (!value) continue

          const messages = Array.isArray(value) ? value : [value]
          const message = messages.map(String).find((msg) => msg.trim().length > 0)

          if (!message) {
            continue
          }

          if (key === "email" || key === "fullName") {
            assignServerError(key, message)
          }
        }
      }

      const message = problem?.detail || problem?.title || "Failed to create user"
      toast.error(message)
      setSubmitError(message)
    }
  })

  return (
    <div className="flex flex-1 flex-col p-6">
      <div className="mb-6 flex flex-col gap-2">
        <div>
          <h1 className="text-2xl font-semibold">Add user</h1>
          <p className="text-muted-foreground mt-2">Provide the user details to invite them into the workspace.</p>
        </div>
        {submitError && (
          <div className="rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive">
            {submitError}
          </div>
        )}
      </div>

      <Form {...form}>
        <form onSubmit={onSubmit} className="space-y-6 max-w-2xl" noValidate>
          <FormField
            control={form.control}
            name="email"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Email</FormLabel>
                <FormControl>
                  <Input
                    type="email"
                    autoComplete="email"
                    placeholder="admin@example.com"
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="fullName"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Full name</FormLabel>
                <FormControl>
                  <Input
                    autoComplete="name"
                    placeholder="Dev Admin"
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className="flex items-center gap-3">
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? "Saving…" : "Create user"}
            </Button>
            <Button
              type="button"
              variant="ghost"
              onClick={() => navigate("/users")}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
          </div>
        </form>
      </Form>
    </div>
  )
}
