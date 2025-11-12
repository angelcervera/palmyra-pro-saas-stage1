Auth domain — handler layer

Implements the generated Chi server interfaces for the auth domain. Parses/validates requests, calls services, and maps errors to RFC7807 ProblemDetails. Keep HTTP-specific logic thin.
