import {useCallback, useEffect, useMemo, useRef, useState} from "react"
import type {ColumnDef} from "@tanstack/react-table"
import type {Users as UsersApi} from "@zengateglobal/api-sdk"
import {
    IconChevronLeft,
    IconChevronRight,
    IconChevronsLeft,
    IconChevronsRight,
    IconLoader,
    IconTrash,
} from "@tabler/icons-react"
import {useLocation, useNavigate} from "react-router-dom"
import {toast} from "sonner"
import {
    Database,
    type DatabaseRow,
    userColumnVisibility,
    userColumns,
} from "@/components/database"
import {Button} from "@/components/ui/button"
import {Label} from "@/components/ui/label"
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from "@/components/ui/select"
import {Skeleton} from "@/components/ui/skeleton"
import {useDeleteUser, useUsersList, type UsersListQuery} from "./use-users"

function formatDate(value?: string) {
    if (!value) return "—"
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return value
    return date.toLocaleDateString(undefined, {
        year: "numeric",
        month: "short",
        day: "numeric",
    })
}

type UsersList = UsersApi.UsersListResponses[200]

export default function UsersPage() {
    const [query, setQuery] = useState<UsersListQuery>({page: 1, pageSize: 10})
    const {data, isLoading, isError, error, isFetching} = useUsersList(query)
    const items = (data?.items ?? []) as UsersList["items"]
    const navigate = useNavigate()
    const location = useLocation()
    const hasShownFlashToast = useRef(false)
    const handleAddUser = useCallback(() => navigate("/users/new"), [navigate])
    useEffect(() => {
        if (isError) {
            const message =
                error instanceof Error ? error.message : "Failed to load users. Please try again."
            toast.error(message)
        }
    }, [isError, error])

    useEffect(() => {
        const state = location.state as
            | {
            toast?: {
                type: "success" | "error" | "info" | "warning"
                message: string
            }
        }
            | undefined

        if (!state?.toast) {
            hasShownFlashToast.current = false
            return
        }

        if (hasShownFlashToast.current) {
            return
        }

        hasShownFlashToast.current = true
        const {type, message} = state.toast

        switch (type) {
            case "success":
                toast.success(message)
                break
            case "error":
                toast.error(message)
                break
            case "warning":
                toast.warning(message)
                break
            default:
                toast.info(message)
                break
        }

        const {toast: _toast, ...rest} = state
        navigate(location.pathname + location.search, {
            replace: true,
            state: Object.keys(rest).length ? rest : null,
        })
    }, [location, navigate])

    const databaseRows = useMemo<DatabaseRow[]>(
        () =>
            items.map((user, index) => ({
                id: index + 1,
                header: user.fullName || user.email,
                type: "User",
                status: user.id ?? "",
                target: user.email ?? "",
                limit: formatDate(user.createdAt),
                reviewer: "",
            })),
        [items],
    )

    const currentPage = data?.page ?? query.page ?? 1
    const pageSize = data?.pageSize ?? query.pageSize ?? 10
    const totalItems = data?.totalItems ?? 0
    const totalPages = data?.totalPages ?? Math.max(1, Math.ceil(totalItems / (pageSize || 1)))

    useEffect(() => {
        if (!data?.totalPages) {
            return
        }
        setQuery((prev) => {
            const targetPage = prev?.page ?? 1
            const safeTotal = data.totalPages || 1
            if (targetPage <= safeTotal) {
                return prev
            }
            return {...prev, page: safeTotal}
        })
    }, [data?.totalPages])

    const columns = useMemo<ColumnDef<DatabaseRow>[]>(() => {
        return [
            ...userColumns,
            {
                id: "actions",
                header: "",
                cell: ({row}) => (
                    <DeleteUserButton
                        userId={String(row.original.status ?? "")}
                        userLabel={row.original.header || row.original.target || "user"}
                    />
                ),
                enableHiding: false,
            },
        ]
    }, [])

    const handlePageChange = useCallback(
        (nextPage: number) => {
            setQuery((prev) => ({...prev, page: Math.max(1, nextPage)}))
        },
        [],
    )

    const handlePageSizeChange = useCallback(
        (nextPageSize: number) => {
            setQuery((prev) => ({
                ...prev,
                page: 1,
                pageSize: nextPageSize,
            }))
        },
        [],
    )

    return (
        <div className="flex flex-1 flex-col p-6">
            <div className="mb-6 flex flex-col gap-4">
                <div>
                    <h1 className="text-2xl font-semibold">Users</h1>
                    <p className="text-muted-foreground mt-2">
                        Overview of platform users. Integrate filters, sorting, and actions as the domain evolves.
                    </p>
                </div>
            </div>

            {isError ? (
                <div className="rounded-md border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
                    Failed to load users. {error instanceof Error ? error.message : "Please try again."}
                </div>
            ) : isLoading ? (
                <div className="divide-border grid gap-4 divide-y p-6">
                    {Array.from({length: 3}).map((_, index) => (
                        <div key={index} className="grid gap-2">
                            <Skeleton className="h-5 w-48"/>
                            <Skeleton className="h-4 w-72"/>
                            <Skeleton className="h-3 w-full"/>
                        </div>
                    ))}
                </div>
            ) : (
                <>
                    <Database
                        data={databaseRows}
                        columns={columns}
                        defaultColumnVisibility={userColumnVisibility}
                        actionLabel="Add user"
                        onAction={handleAddUser}
                        hidePaginationControls
                        pageSize={databaseRows.length > 0 ? databaseRows.length : undefined}
                    />
                    <UsersPagination
                        className="mt-6"
                        page={currentPage}
                        pageSize={pageSize}
                        totalItems={totalItems}
                        totalPages={totalPages}
                        onPageChange={handlePageChange}
                        onPageSizeChange={handlePageSizeChange}
                        disabled={isLoading || isFetching}
                    />
                </>
            )}
        </div>
    )
}

function DeleteUserButton({userId, userLabel}: { userId: string; userLabel: string }) {
    const {mutateAsync, isPending} = useDeleteUser()

    const handleDelete = useCallback(async () => {
        const trimmedId = userId?.trim()
        if (!trimmedId) {
            toast.error("Unable to delete this user. The user identifier is missing.")
            return
        }

        const confirmation = window.confirm(
            `Delete ${userLabel || "this user"}? This action cannot be undone.`,
        )
        if (!confirmation) {
            return
        }

        try {
            await mutateAsync(trimmedId)
            toast.success(`${userLabel || "User"} deleted.`)
        } catch (err) {
            const message =
                err instanceof Error ? err.message : "Failed to delete user. Please try again."
            toast.error(message)
        }
    }, [mutateAsync, userId, userLabel])

    return (
        <Button
            variant="ghost"
            size="icon"
            className="text-destructive hover:text-destructive focus-visible:text-destructive"
            onClick={handleDelete}
            disabled={isPending}
        >
            {isPending ? (
                <IconLoader className="animate-spin" size={16} aria-hidden="true"/>
            ) : (
                <IconTrash size={16} aria-hidden="true"/>
            )}
            <span className="sr-only">Delete user</span>
        </Button>
    )
}

type UsersPaginationProps = {
    page: number
    pageSize: number
    totalItems: number
    totalPages: number
    onPageChange: (page: number) => void
    onPageSizeChange: (pageSize: number) => void
    disabled?: boolean
    className?: string
}

function UsersPagination({
                              page,
                              pageSize,
                              totalItems,
                              totalPages,
                              onPageChange,
                              onPageSizeChange,
                              disabled = false,
                              className = "",
                          }: UsersPaginationProps) {
    const safePageSize = pageSize > 0 ? pageSize : 10
    const safeTotalPages = Math.max(totalPages, 1)
    const currentPage = Math.min(Math.max(page, 1), safeTotalPages)
    const hasPrevious = currentPage > 1
    const hasNext = currentPage < safeTotalPages
    const from = totalItems === 0 ? 0 : (currentPage - 1) * safePageSize + 1
    const to = totalItems === 0 ? 0 : Math.min(totalItems, currentPage * safePageSize)

    const handleFirst = useCallback(() => {
        if (hasPrevious) {
            onPageChange(1)
        }
    }, [hasPrevious, onPageChange])

    const handlePrev = useCallback(() => {
        if (hasPrevious) {
            onPageChange(currentPage - 1)
        }
    }, [hasPrevious, currentPage, onPageChange])

    const handleNext = useCallback(() => {
        if (hasNext) {
            onPageChange(currentPage + 1)
        }
    }, [hasNext, currentPage, onPageChange])

    const handleLast = useCallback(() => {
        if (hasNext) {
            onPageChange(safeTotalPages)
        }
    }, [hasNext, onPageChange, safeTotalPages])

    return (
        <div className={`flex flex-col gap-4 border-t border-border pt-4 lg:flex-row lg:items-center lg:justify-between ${className}`}>
            <div className="text-muted-foreground text-sm">
                {totalItems === 0 ? (
                    "No users found."
                ) : (
                    <>Showing {from}-{to} of {totalItems} users</>
                )}
            </div>
            <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:gap-4">
                <div className="flex items-center gap-2">
                    <Label htmlFor="users-rows-per-page" className="text-sm font-medium">
                        Rows per page
                    </Label>
                    <Select
                        value={`${safePageSize}`}
                        onValueChange={(value) => onPageSizeChange(Number(value))}
                        disabled={disabled}
                    >
                        <SelectTrigger
                            size="sm"
                            className="w-24"
                            id="users-rows-per-page"
                        >
                            <SelectValue placeholder={safePageSize}/>
                        </SelectTrigger>
                        <SelectContent side="top">
                            {[10, 20, 30, 40, 50].map((size) => (
                                <SelectItem key={size} value={`${size}`}>
                                    {size}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>
            </div>
            <div className="flex items-center gap-2 lg:ml-auto">
                <span className="text-muted-foreground text-sm">
                    Page {currentPage} of {safeTotalPages}
                </span>
                <Button
                    variant="outline"
                    size="icon"
                    className="hidden h-8 w-8 p-0 lg:flex"
                    onClick={handleFirst}
                    disabled={!hasPrevious || disabled}
                >
                    <span className="sr-only">Go to first page</span>
                    <IconChevronsLeft size={18}/>
                </Button>
                <Button
                    variant="outline"
                    size="icon"
                    className="h-8 w-8"
                    onClick={handlePrev}
                    disabled={!hasPrevious || disabled}
                >
                    <span className="sr-only">Go to previous page</span>
                    <IconChevronLeft size={18}/>
                </Button>
                <Button
                    variant="outline"
                    size="icon"
                    className="h-8 w-8"
                    onClick={handleNext}
                    disabled={!hasNext || disabled}
                >
                    <span className="sr-only">Go to next page</span>
                    <IconChevronRight size={18}/>
                </Button>
                <Button
                    variant="outline"
                    size="icon"
                    className="hidden h-8 w-8 lg:flex"
                    onClick={handleLast}
                    disabled={!hasNext || disabled}
                >
                    <span className="sr-only">Go to last page</span>
                    <IconChevronsRight size={18}/>
                </Button>
            </div>
        </div>
    )
}
